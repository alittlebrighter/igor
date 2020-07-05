package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/nats-io/nats.go"
	"github.com/robertkrimen/otto"

	"github.com/alittlebrighter/igor"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

const (
	ReducerDir = "reducers"
	StateDir   = "state"
)

func main() {
	nc, err := nats.Connect(nats.DefaultURL)
	handleErr(err, true)
	defer nc.Close()
	ec, err := nats.NewEncodedConn(nc, "json")
	handleErr(err, true)
	defer ec.Close()

	// initialize the starting state?
	// read from state directory

	// both effects and reducers are stored in respective directories
	// effects
	// at start read in all effects and reducers files and setup subscriptions based on annotations `"igor.actionSubscriptions "action1","action2",...`
	// reducers
	// reducerSubs: keys=action types, values=list of reducers subscribed
	vm := otto.New()

	reducerSubs := ProcessReducers(vm, ReducerDir, StateDir)

	// listen for incoming actions
	events := make(chan igor.Event, 10)
	eventSub, err := ec.BindRecvQueueChan("igor.events", "events_queue", events)
	handleErr(err, true)
	defer eventSub.Unsubscribe()
	defer close(events)

	// publish new state
	updater := func(path string, update interface{}) {
		pathParts := strings.Split(path, "/")
		pathParts[len(pathParts)-1] = strings.Split(pathParts[len(pathParts)-1], ".")[0]
		ec.Publish("igor."+StateDir+strings.Join(pathParts, "."), update)
	}

	go HandleEvents(events, updater, vm, reducerSubs, StateDir)

	// load device controllers via go modules
	// each module should have two methods, StateSubscription() string and func([]byte)
	// device controllers run code based on new state dispatching error events as needed

}

func handleErr(err error, fatal bool) {
	if err == nil {
		return
	}

	fmt.Println("error:", err)

	if fatal {
		os.Exit(1)
	}
}

func ProcessReducers(vm *otto.Otto, reducerDir, stateDir string) map[igor.EventType][]igor.OttoScript {
	reducerSubs := map[igor.EventType][]igor.OttoScript{
		// this is igor.wildcard
		igor.EventWildcard: []igor.OttoScript{},
	}

	filepath.Walk(reducerDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		program, eTypes, err := ParseScript(vm, path)
		if err != nil {
			handleErr(err, false)
			return err
		}

		for _, sub := range eTypes {
			event := igor.EventType(sub)
			if _, exists := reducerSubs[event]; !exists {
				reducerSubs[event] = []igor.OttoScript{}
			}

			partialPath := strings.Replace(path, reducerDir, "", 1)
			reducerSubs[event] = append(reducerSubs[event],
				igor.OttoScript{Path: partialPath, Program: program})

			os.MkdirAll(stateDir+"/"+strings.TrimSuffix(partialPath, info.Name()), 0700)
		}

		return nil
	})

	return reducerSubs
}

// HandleEvents has way too many arguments
func HandleEvents(eventsIn <-chan igor.Event, update func(string, interface{}), vm *otto.Otto, reducers map[igor.EventType][]igor.OttoScript, stateDir string) {
	for event := range eventsIn {
		for _, script := range append(reducers[event.Type], reducers["*"]...) {
			state := make(map[string]interface{})

			// reducer is reducers/**/script.js and state is state/**/state.json
			statePath := stateDir + script.Path + "on"
			stateData, err := ioutil.ReadFile(statePath)
			if err == nil {
				json.Unmarshal(stateData, state)
			}

			vm.Set("input", map[string]interface{}{
				"state": state,
				"event": event,
			})

			result, err := vm.Run(script.Program)
			if err != nil {
				fmt.Println("error running:", err.Error())
			}

			exported, err := result.Export()
			handleErr(err, false)
			if err != nil || exported == nil {
				continue
			}

			newState, err := json.Marshal(exported)
			handleErr(err, false)
			if err != nil {
				continue
			}

			// publish to NATS, a separate subscriber should write the resulting state
			update(script.Path[1:], newState)
			handleErr(err, false)
		}
	}
}

// ParseScript returns the source contents and event type subscriptions or any errors
func ParseScript(vm *otto.Otto, scriptPath string) (program *otto.Script, subs []string, err error) {
	script, err := os.Open(scriptPath)
	if err != nil {
		return nil, nil, err
	}

	var src string
	var subsFound bool
	prefix := `"igor.subs`

	scriptReader := bufio.NewReader(script)
	for {
		line, err := scriptReader.ReadString(byte('\n'))
		if err != nil {
			break
		}

		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			subs = igor.ParseSubscriptionDirective(trimmed, prefix)
			subsFound = true
		}
		if len(trimmed) > 0 {
			src = src + line
		}
	}

	if !subsFound {
		err = errors.New("GetSubscriptions: no subscription directive found")
		return
	}

	program, err = vm.Compile(scriptPath, nil)
	if err != nil {
		return
	}

	return
}

type StateUpdate struct {
	Path   string
	Update interface{}
}

/*
distributed issues:
heartbeat events
who is the leader processing events? solved with NATS queuing
*/
