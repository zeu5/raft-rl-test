package types

// Trace of an episode as triplets (state, action, nextState)
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

func (t *Trace) Slice(from, to int) *Trace {
	slicedTrace := NewTrace()
	for i := from; i < to; i++ {
		slicedTrace.Append(i-from, t.states[i], t.actions[i], t.nextStates[i])
	}
	return slicedTrace
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

func (t *Trace) Last() (State, Action, State, bool) {
	if len(t.states) < 1 {
		return nil, nil, nil, false
	}
	lastIndex := len(t.states) - 1
	return t.states[lastIndex], t.actions[lastIndex], t.nextStates[lastIndex], true
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
