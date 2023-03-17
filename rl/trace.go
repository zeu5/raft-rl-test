package rl

type Trace struct {
	states     []State
	actions    []Action
	nextStates []State
}

func NewTrace() *Trace {
	return &Trace{
		states:     make([]State, 0),
		actions:    make([]Action, 0),
		nextStates: make([]State, 0),
	}
}

func (t *Trace) Append(step int, state State, action Action, nextState State) {
	t.states = append(t.states, state)
	t.actions = append(t.actions, action)
	t.nextStates = append(t.nextStates, nextState)
}

func (t *Trace) Len() int {
	return len(t.states)
}

func (t *Trace) Get(i int) (State, Action, State, bool) {
	if i >= len(t.states) {
		return nil, nil, nil, false
	}
	return t.states[i], t.actions[i], t.nextStates[i], true
}

func (t *Trace) GetPrefix(i int) (*Trace, bool) {
	if i > len(t.states) {
		return nil, false
	}
	return &Trace{
		states:     t.states[0:i],
		actions:    t.actions[0:i],
		nextStates: t.nextStates[0:i],
	}, true
}
