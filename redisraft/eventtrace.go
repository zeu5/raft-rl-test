package redisraft

import (
	"encoding/json"
	"sync"
)

type Event struct {
	Name   string
	Node   int `json:"-"`
	Params map[string]interface{}
	Reset  bool
}

func (e Event) Copy() Event {
	new := Event{
		Name:   e.Name,
		Node:   e.Node,
		Params: make(map[string]interface{}),
		Reset:  e.Reset,
	}
	for k, v := range e.Params {
		new.Params[k] = v
	}
	return new
}

type EventTrace struct {
	Events []Event
	lock   *sync.Mutex
}

var _ json.Marshaler = &EventTrace{}

func NewEventTrace() *EventTrace {
	return &EventTrace{
		Events: make([]Event, 0),
		lock:   new(sync.Mutex),
	}
}

func (e *EventTrace) Copy() *EventTrace {
	e.lock.Lock()
	defer e.lock.Unlock()
	new := &EventTrace{
		Events: make([]Event, len(e.Events)),
		lock:   new(sync.Mutex),
	}
	for i, e := range e.Events {
		new.Events[i] = e.Copy()
	}
	return new
}

func (et *EventTrace) Add(e Event) {
	et.lock.Lock()
	defer et.lock.Unlock()
	et.Events = append(et.Events, e.Copy())
}

func (et *EventTrace) Reset() {
	et.lock.Lock()
	defer et.lock.Unlock()
	et.Events = make([]Event, 0)
}

func (et *EventTrace) MarshalJSON() ([]byte, error) {
	et.lock.Lock()
	defer et.lock.Unlock()
	bs, err := json.Marshal(et.Events)
	return bs, err
}
