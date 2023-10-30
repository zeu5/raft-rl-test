package types

import (
	"encoding/json"
	"os"
)

// Trace of an episode as triplets (state, action, nextState)
type Trace struct {
	states     []State
	actions    []Action
	nextStates []State
	rewards    []bool
}

func NewTrace() *Trace {
	return &Trace{
		states:     make([]State, 0),
		actions:    make([]Action, 0),
		nextStates: make([]State, 0),
		rewards:    make([]bool, 0),
	}
}

func (t *Trace) Slice(from, to int) *Trace {
	slicedTrace := NewTrace()
	for i := from; i < to; i++ {
		slicedTrace.AppendWithReward(i-from, t.states[i], t.actions[i], t.nextStates[i], t.rewards[i])
	}
	return slicedTrace
}

func (t *Trace) Append(step int, state State, action Action, nextState State) {
	t.states = append(t.states, state)
	t.actions = append(t.actions, action)
	t.nextStates = append(t.nextStates, nextState)
	t.rewards = append(t.rewards, false)
}

func (t *Trace) AppendWithReward(step int, state State, action Action, nextState State, reward bool) {
	t.states = append(t.states, state)
	t.actions = append(t.actions, action)
	t.nextStates = append(t.nextStates, nextState)
	t.rewards = append(t.rewards, reward)
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

func (t *Trace) GetWithReward(i int) (State, Action, State, bool, bool) {
	if i >= len(t.states) {
		return nil, nil, nil, false, false
	}
	return t.states[i], t.actions[i], t.nextStates[i], t.rewards[i], true
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
		rewards:    t.rewards[0:i],
	}, true
}

func (t *Trace) Record(p string) {
	out := make(map[string]interface{})
	out["states"] = t.states
	out["actions"] = t.actions
	out["next_states"] = t.nextStates
	out["rewards"] = t.rewards

	bs, err := json.Marshal(out)
	if err != nil {
		return
	}
	os.WriteFile(p, bs, 0644)
}
