package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"plugin"
	"strings"
	"syscall"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/nats-io/nats.go"
	"github.com/robertkrimen/otto"

	"github.com/alittlebrighter/igor"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

const (
	StoreDir = "state"

	ChannelPrefix   = "igor"
	EventStream     = ChannelPrefix + ".events"
	EventQueueGroup = "igor_events_queue"
)

func main() {
	// NATS is crucial to all of the operations
	// TODO: interact with NATS through an interface
	nc, err := nats.Connect(nats.DefaultURL)
	handleErr(err, true)
	defer nc.Close()
	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	handleErr(err, true)
	defer ec.Close()

	// both effects and reducers are stored in respective directories
	// TODO: effects (read in actions and asynchronously dispatch 0:n actions)
	// at start read in all effects and reducers files and setup subscriptions based on annotations `"igor.actionSubscriptions "action1","action2",...`
	// reducers
	// reducerSubs: keys=action types, values=list of reducers subscribed
	vm := otto.New()

	reducers := make(map[igor.EventType][]igor.OttoScript)
	components := map[string]igor.IgorPlugin{}
	err = filepath.Walk(StoreDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		directory := path[0:strings.LastIndex(path, "/")]

		switch {
		case strings.HasSuffix(path, ".js"): // TODO: make this case conditional a method to accommodate more reducer types
			program, eTypes, err := ParseScript(vm, path)
			if err != nil {
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
			components[igor.FilePathToTopic(directory)] = newComponent
		}

		return nil
	})
	handleErr(err, true)

	// initialize the starting state
	// read from state directory
	initialState, err := igor.FilesToJson(StoreDir)
	handleErr(err, true)
	state := igor.NewAutomationState(initialState)

	stateUpdates := make(chan *nats.Msg, 10)
	stateSub, err := nc.ChanSubscribe(ChannelPrefix+"."+StoreDir+".>", stateUpdates)
	defer stateSub.Unsubscribe()
	defer close(stateUpdates)
	go func(updates <-chan *nats.Msg, listeners map[string]igor.IgorPlugin, global *igor.AutomationState) {
		for update := range updates {
			updateAddress := strings.Split(update.Subject, ".")
			global.Mutate(update.Data, updateAddress...)

			for sub, listener := range listeners {
				topic := strings.TrimPrefix(update.Subject, ChannelPrefix+".")
				if strings.HasPrefix(topic, sub) || topic == StoreDir+".INIT" {
					listener.UpdateState(updateAddress, update.Data)
				}
			}
		}
	}(stateUpdates, components, state)

	nc.Publish(strings.Join([]string{ChannelPrefix, StoreDir, "INIT"}, "."), state.State())

	// listen for incoming actions
	events := make(chan igor.Event, 10)
	eventSub, err := ec.BindRecvQueueChan(EventStream, EventQueueGroup, events)
	handleErr(err, true)
	defer eventSub.Unsubscribe()
	defer close(events)

	// publish new state
	updater := func(path []string, update []byte) {
		fmt.Println("updating", strings.Join(append([]string{ChannelPrefix, StoreDir}, path...), "."))
		nc.Publish(strings.Join(append([]string{ChannelPrefix, StoreDir}, path...), "."), update)
	}

	go HandleEvents(events, updater, vm, reducers, state)

	nc.Publish(EventStream, []byte(`{"type":"test","payload":{"door":"1"}}`))

	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
}

func handleErr(err error, fatal bool) {
	if err == nil {
		return
	}
	panic(err)
	/*
		fmt.Println("error:", err)

		if fatal {
			os.Exit(1)
		}
	*/
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
func HandleEvents(eventsIn <-chan igor.Event, update func([]string, []byte), vm *otto.Otto, reducers map[igor.EventType][]igor.OttoScript, state *igor.AutomationState) {
	for event := range eventsIn {
		for _, script := range append(reducers[event.Type], reducers[igor.EventWildcard]...) {
			current := state.State()
			if event.Type != igor.EventWildcard {
				statePath := strings.TrimPrefix(script.Path[:strings.LastIndex(script.Path, "/")], StoreDir+"/")
				current, _ = state.Select(strings.Split(statePath, "/")...)
			}

			vm.Set("input", map[string]interface{}{
				"state": current,
				"event": event,
			})

			result, err := vm.Run(script.Program)
			if err != nil {
				handleErr(err, false)
				continue
			}

			// deltaPath should be the path within the state structure where changes were made,
			// empty string denotes the entire state should be swapped
			deltaPathStr, err := result.Object().Get("path")
			// delta is the new state the should be applied at the specified path
			delta, err := result.Object().Get("delta")
			if err != nil {
				handleErr(err, false)
				continue
			}

			deltaPath, err := deltaPathStr.ToString()
			exported, err := delta.Export()
			if err != nil {
				handleErr(err, false)
				continue
			}

			newState, err := json.Marshal(exported)
			if err != nil {
				handleErr(err, false)
				continue
			}

			filePath := strings.TrimPrefix(script.Path, StoreDir+"/")
			filePath = filePath[:strings.LastIndex(filePath, "/")]
			updatePath := append(
				igor.TrimStringSlice(strings.Split(filePath, "/")),
				igor.TrimStringSlice(strings.Split(deltaPath, "."))...,
			)

			// publish to NATS, a separate subscriber should write the resulting state
			update(updatePath, newState)
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

	init := componentInit.(func(func(igor.Event), []string) igor.IgorPlugin)
	return init(publisher, []string{componentPath[0:strings.LastIndex(componentPath, "/")]}), nil
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
