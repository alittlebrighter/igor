package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"plugin"
	"strings"
	"time"

	"github.com/alittlebrighter/embd"
	jsoniter "github.com/json-iterator/go"
	"github.com/nats-io/nats.go"
	"github.com/robertkrimen/otto"

	"github.com/alittlebrighter/igor"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

const (
	// should we just have one file structure instead of forcing duplication of a folder hierarchy
	// igor/state/garageDoors/{state.json,reducer.js,garageDoors.so}
	ReducerDir = "reducers"
	StoreDir   = "state"

	ChannelPrefix = "igor."
	EventStream   = ChannelPrefix + "events"
)

func main() {
	// NATS is crucial to all of the operations
	// TODO: interact with NATS through an interface
	nc, err := nats.Connect(nats.DefaultURL)
	handleErr(err, true)
	defer nc.Close()
	ec, err := nats.NewEncodedConn(nc, "json")
	handleErr(err, true)
	defer ec.Close()

	// start any component plugins
	if err := embd.InitGPIO(); err != nil {
		panic(err)
	}
	defer embd.CloseGPIO()

	// both effects and reducers are stored in respective directories
	// TODO: effects (read in actions and asynchronously dispatch 0:n actions)
	// at start read in all effects and reducers files and setup subscriptions based on annotations `"igor.actionSubscriptions "action1","action2",...`
	// reducers
	// reducerSubs: keys=action types, values=list of reducers subscribed
	vm := otto.New()

	states := make(map[string][]byte)
	reducers := make(map[igor.EventType][]igor.OttoScript)
	components := map[string]igor.IgorPlugin{}
	filepath.Walk(ComponentDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		directory := path[0:strings.LastIndex(path, "/")]

		switch {
		case strings.HasSuffix(path, ".js"): // reducers, TODO: make this case conditional a method to accommodate more reducer types
			program, eTypes, err := ParseScript(vm, path)
			if err != nil {
				handleErr(err, false)
				return err
			}

			for _, sub := range eTypes {
				event := igor.EventType(sub)
				if _, exists := reducers[event]; !exists {
					reducers[event] = []igor.OttoScript{}
				}

				reducers[event] = append(reducers[event],
					igor.OttoScript{Path: path, Program: program})

			}
		case strings.HasSuffix(path, ".so"): // components that sense or control things, not necessarily present in every directory
			newComponent, err := ProcessComponent(path, DispatcherFactory(ec))
			if err != nil {
				return err
			}
			components[directory] = newComponent
		case strings.HasSuffix(path, ".json"): // existing state, not guaranteed to be in every directory
			partialState, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			states[directory] = partialState
		default:
			return nil
		}

		return nil
	})

	// initialize the starting state
	// read from state directory

	// listen for incoming actions
	events := make(chan igor.Event, 10)
	eventSub, err := ec.BindRecvQueueChan(EventStream, "events_queue", events)
	handleErr(err, true)
	defer eventSub.Unsubscribe()
	defer close(events)

	// publish new state
	updater := func(path string, update interface{}) {
		pathParts := strings.Split(path, "/")
		pathParts[len(pathParts)-1] = strings.Split(pathParts[len(pathParts)-1], ".")[0]
		ec.Publish(ChannelPrefix+StoreDir+strings.Join(pathParts, "."), update)
	}

	go HandleEvents(events, updater, vm, reducers, StoreDir)

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
	reducerSubs := map[igor.EventType][]igor.OttoScript{}

	filepath.Walk(reducerDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		return nil
	})

	return reducerSubs
}

// HandleEvents has way too many arguments
func HandleEvents(eventsIn <-chan igor.Event, update func(string, interface{}), vm *otto.Otto, reducers map[igor.EventType][]igor.OttoScript, stateDir string) {
	for event := range eventsIn {
		for _, script := range append(reducers[event.Type], reducers[igor.EventWildcard]...) {
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

func ProcessComponent(componentPath string, publisher func(igor.Event)) (igor.IgorPlugin, error) {
	p, err := plugin.Open(componentPath)
	if err != nil {
		return nil, err
	}

	componentInit, err := p.Lookup(igor.PluginInitSymbol)
	if err != nil {
		return nil, err
	}

	return componentInit.(func(func(igor.Event), []string) igor.IgorPlugin)(publisher, []string{componentPath[0:strings.LastIndex(componentPath, "/")]}), nil
}

type StateUpdate struct {
	Path   string
	Update interface{}
}

type Dispatcher = func(igor.Event)

func DispatcherFactory(natsConn *nats.EncodedConn) Dispatcher {
	return func(e igor.Event) {
		e.Timestamp = time.Now()
		natsConn.Publish(EventStream, e)
	}
}

/*
distributed issues:
heartbeat events
who is the leader processing events? solved with NATS queuing
*/
