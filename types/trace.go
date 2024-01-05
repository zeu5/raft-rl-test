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
	if t == nil {
		return 0
	}
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

func (t *Trace) MarshalJSON() ([]byte, error) {
	out := make(map[string]interface{})
	states := make([]map[string]interface{}, len(t.states))
	for i, s := range t.states {
		states[i] = map[string]interface{}{
			"key":   s.Hash(),
			"state": s,
		}
	}
	out["states"] = states
	actions := make([]map[string]interface{}, len(t.actions))
	for i, a := range t.actions {
		actions[i] = map[string]interface{}{
			"key":    a.Hash(),
			"action": a,
		}
	}
	out["actions"] = actions
	nextStates := make([]map[string]interface{}, len(t.states))
	for i, s := range t.nextStates {
		nextStates[i] = map[string]interface{}{
			"key":   s.Hash(),
			"state": s,
		}
	}
	out["next_states"] = nextStates
	out["rewards"] = t.rewards
	return json.Marshal(out)
}

func (t *Trace) Record(p string) {
	bs, err := json.Marshal(t)
	if err != nil {
		return
	}
	os.WriteFile(p, bs, 0644)
}

// func (t *Trace) RecordReadable(p string) {
// 	init := "### TRACE START ###\n"

// 	body := ""
// 	for i := range len(t.states) {
// 		body = fmt.Sprintf("%s\n ---state %d--- \n", body, i)
// 		pState := state.(*Partition)
// 		replicaStates := pState.
// 	}

// 	end := "### TRACE END ###\n"
// 	os.WriteFile(p, bs, 0644)
// }

// Trace used by RM (state, action, nextState, reward, outOfSpace)
type RmTrace struct {
	states     []State
	actions    []Action
	nextStates []State
	rewards    []bool
	outOfSpace []bool
}

func NewRmTrace() *RmTrace {
	return &RmTrace{
		states:     make([]State, 0),
		actions:    make([]Action, 0),
		nextStates: make([]State, 0),
		rewards:    make([]bool, 0),
		outOfSpace: make([]bool, 0),
	}
}

func (t *RmTrace) Slice(from, to int) *RmTrace {
	slicedTrace := NewRmTrace()
	for i := from; i < to; i++ {
		slicedTrace.Append(i-from, t.states[i], t.actions[i], t.nextStates[i], t.rewards[i], t.outOfSpace[i])
	}
	return slicedTrace
}

func (t *RmTrace) Append(step int, state State, action Action, nextState State, reward bool, outOfSpace bool) {
	t.states = append(t.states, state)
	t.actions = append(t.actions, action)
	t.nextStates = append(t.nextStates, nextState)
	t.rewards = append(t.rewards, reward)
	t.outOfSpace = append(t.outOfSpace, outOfSpace)
}

func (t *RmTrace) Len() int {
	return len(t.states)
}

func (t *RmTrace) Get(i int) (State, Action, State, bool, bool, bool) {
	if i >= len(t.states) {
		return nil, nil, nil, false, false, false
	}
	return t.states[i], t.actions[i], t.nextStates[i], t.rewards[i], t.outOfSpace[i], true
}

func (t *RmTrace) Last() (State, Action, State, bool, bool, bool) {
	if len(t.states) < 1 {
		return nil, nil, nil, false, false, false
	}
	lastIndex := len(t.states) - 1
	return t.states[lastIndex], t.actions[lastIndex], t.nextStates[lastIndex], t.rewards[lastIndex], t.outOfSpace[lastIndex], true
}

func (t *RmTrace) GetPrefix(i int) (*RmTrace, bool) {
	if i > len(t.states) {
		return nil, false
	}
	return &RmTrace{
		states:     t.states[0:i],
		actions:    t.actions[0:i],
		nextStates: t.nextStates[0:i],
		rewards:    t.rewards[0:i],
		outOfSpace: t.outOfSpace[0:i],
	}, true
}
