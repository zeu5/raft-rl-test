package explorer

import "encoding/json"

type State struct {
	Key   string
	State map[string]interface{}
}

func (s *State) String() string {
	bs, _ := json.Marshal(s)
	return string(bs)
}

type Action struct {
	Key    string
	Action interface{}
}

func (a *Action) String() string {
	return a.Key
}

type Trace struct {
	States     []*State
	Actions    []*Action
	NextStates []*State
	Rewards    []bool
}

func NewTrace() *Trace {
	return &Trace{
		States:     make([]*State, 0),
		Actions:    make([]*Action, 0),
		NextStates: make([]*State, 0),
		Rewards:    make([]bool, 0),
	}
}

func (t *Trace) Len() int {
	return len(t.States)
}

func (t *Trace) Get(index int) (*State, *Action, *State, bool) {
	if index > len(t.States) {
		return nil, nil, nil, false
	}
	return t.States[index], t.Actions[index], t.NextStates[index], t.Rewards[index]
}
