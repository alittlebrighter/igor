package igor

import (
	"encoding/json"
	"sync"

	"github.com/buger/jsonparser"
)

type AutomationState struct {
	state []byte
	sync.RWMutex
}

func NewAutomationState(init []byte) *AutomationState {
	initState := []byte("{}")
	if init != nil && IsJSON(init) {
		initState = init
	}

	return &AutomationState{
		state: initState,
	}
}

func (as *AutomationState) State() []byte {
	as.RLock()
	defer as.RUnlock()

	return as.state
}

func (as *AutomationState) Select(path ...string) ([]byte, error) {
	as.RLock()
	defer as.RUnlock()

	val, _, _, err := jsonparser.Get(as.state, path...)
	return val, err
}

func (as *AutomationState) Mutate(update []byte, updatePath ...string) error {
	as.Lock()
	defer as.Unlock()

	var err error
	as.state, err = jsonparser.Set(as.state, update, updatePath...)
	return err
}

func IsJSON(data []byte) bool {
	var js json.RawMessage
	return json.Unmarshal(data, &js) == nil
}
