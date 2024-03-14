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
	// Additional info related to each step
	additionalInfo []map[string]interface{}
}

func NewTrace() *Trace {
	return &Trace{
		states:         make([]State, 0),
		actions:        make([]Action, 0),
		nextStates:     make([]State, 0),
		additionalInfo: make([]map[string]interface{}, 0),
	}
}

func (t *Trace) Slice(from, to int) *Trace {
	slicedTrace := NewTrace()
	for i := from; i < to; i++ {
		slicedTrace.Append(i-from, t.states[i], t.actions[i], t.nextStates[i], t.additionalInfo[i])
	}
	return slicedTrace
}

func (t *Trace) AppendCtx(sCtx *StepContext) {
	addInfo := make(map[string]interface{})
	if sCtx.addInfo != nil {
		for k, v := range sCtx.addInfo {
			addInfo[k] = v
		}
	}
	t.Append(sCtx.Step, sCtx.State, sCtx.Action, sCtx.NextState, addInfo)
}

func (t *Trace) Append(step int, state State, action Action, nextState State, addInfo map[string]interface{}) {
	t.states = append(t.states, state)
	t.actions = append(t.actions, action)
	t.nextStates = append(t.nextStates, nextState)
	t.additionalInfo = append(t.additionalInfo, addInfo)
}
func (t *Trace) Len() int {
	if t == nil {
		return 0
	}
	return len(t.states)
}

// Get returns the state, action, next state and a boolean indicating if the index is valid
func (t *Trace) Get(i int) (State, Action, State, bool) {
	if i >= len(t.states) {
		return nil, nil, nil, false
	}
	return t.states[i], t.actions[i], t.nextStates[i], true
}

func (t *Trace) GetAdditionalInfo(step int) (map[string]interface{}, bool) {
	if step >= len(t.states) {
		return nil, false
	}
	return t.additionalInfo[step], t.additionalInfo[step] != nil
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
		states:         t.states[0:i],
		actions:        t.actions[0:i],
		nextStates:     t.nextStates[0:i],
		additionalInfo: t.additionalInfo[0:i],
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
	out["additionalInfo"] = t.additionalInfo
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
