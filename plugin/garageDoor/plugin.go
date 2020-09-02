package main

import (
	"errors"
	"time"

	"github.com/buger/jsonparser"
	jsoniter "github.com/json-iterator/go"

	"github.com/alittlebrighter/embd"
	"github.com/alittlebrighter/igor"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func Init(dispatcher func(igor.Event), statePath []string) igor.IgorPlugin {
	return NewGarageDoorsController(dispatcher, statePath)
}

type GarageDoorsController struct {
	Config      *GarageDoorsConfig `json:"config"`
	LastTrigger struct {
		Time  int64 `json:"time"`
		Force bool  `json:"force"`
	} `json:"lastTrigger"`
	dispatcher func(igor.Event)
	statePath  []string
}

type GarageDoorsConfig struct {
	DoorMap map[string]*GarageDoor `json:"doorMap"`
	Trigger time.Duration          `json:"trigger"`
	Force   time.Duration          `json:"force"`
}

func NewGarageDoorsController(dispatcher func(igor.Event), statePath []string) *GarageDoorsController {
	return &GarageDoorsController{dispatcher: dispatcher, statePath: statePath}
}

func (gdc *GarageDoorsController) UpdateState(path []string, update []byte) error {
	switch {
	case len(path) <= 1:
		update, _, _, _ = jsonparser.Get(update, gdc.statePath...)
		fallthrough
	case len(path) > len(gdc.statePath) && path[len(gdc.statePath)-1] == "config":
		// update config
		return gdc.SetConfigFromData(update)
	case len(path) > len(gdc.statePath) && path[len(gdc.statePath)-1] == "lastTrigger":
		// trigger door
		door, err := jsonparser.GetString(update, []string{"door"}...)
		force, err := jsonparser.GetBoolean(update, []string{"force"}...)
		if err != nil {
			return err
		}
		return gdc.Trigger(door, force)
	default:
		// update config
	}
	return nil
}

func (gdc *GarageDoorsController) SetConfigFromData(newConfig []byte) error {
	return json.Unmarshal(newConfig, gdc.config)
}

func (gdc *GarageDoorsController) Trigger(doorName string, force bool) error {
	door, exists := gdc.config.DoorMap[doorName]
	if !exists {
		// dispatch door not found event
		return errors.New("no door found with label '" + doorName + "'")
	}

	triggerTime := gdc.config.Trigger
	if force {
		triggerTime = gdc.config.Force
	}
	door.Trigger(triggerTime)

	return nil
}

type GarageDoor struct {
	trigger embd.DigitalPin
	cancel  chan struct{}
}

func NewGarageDoor(bcm int) (*GarageDoor, error) {
	pin, err := embd.NewDigitalPin(bcm)
	if err != nil {
		return nil, err
	}
	pin.SetDirection(embd.Out)
	return &GarageDoor{trigger: pin, cancel: make(chan struct{})}, nil
}

func (gd *GarageDoor) Trigger(holdFor time.Duration) {
	if state, _ := gd.trigger.Read(); state == embd.High {
		gd.cancel <- struct{}{}
	} else {
		gd.trigger.Write(embd.High)
		timer := time.NewTimer(holdFor)
		go func() {
			select {
			case <-timer.C:
			case <-gd.cancel:
				timer.Stop()
			}
			gd.trigger.Write(embd.Low)
		}()
	}
}
