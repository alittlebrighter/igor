package igor

import (
	"strings"
	"time"

	"github.com/robertkrimen/otto"
)

type EventType string

const (
	EventWildcard EventType = "*"
)

type Store interface {
	Dispatch(Event)
	Subscribe() interface{} // state
}

type Event struct {
	Type      EventType   `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

type Runnable interface {
	Run(args interface{}) interface{}
}

// parses a directive line inside of a JS file
func ParseSubscriptionDirective(directive, prefix string) []string {
	const wildcard = "*"

	directive = strings.TrimSpace(directive)
	directive = strings.TrimPrefix(directive, prefix)
	directive = strings.TrimSuffix(directive, `";`)

	// map as an easy way to dedupe
	eventMap := make(map[string]struct{})
	events := strings.Split(directive, ",")
	for _, rawType := range events {
		eventMap[strings.Trim(strings.TrimSpace(rawType), `"'`)] = struct{}{}
	}

	if _, exists := eventMap[wildcard]; exists {
		return []string{wildcard}
	} else {
		toReturn := []string{}
		for eType := range eventMap {
			toReturn = append(toReturn, eType)
		}
		return toReturn
	}
}

type OttoScript struct {
	Path          string
	Program       *otto.Script
	ResultHandler func(interface{})
}

type ScriptStore interface {
	Fetch()
}

type ScriptManager interface {
	HandleEvents(<-chan Event)
}

func NewScriptRunner(dirs []string)
