package policies

import "github.com/zeu5/raft-rl-test/types"

type StateAction func(types.State, []types.Action) (types.Action, bool)

type IfThenStateAction struct {
	If func(types.State) bool
	T  func([]types.Action) (types.Action, bool)
}

func If(cond func(types.State) bool) *IfThenStateAction {
	return &IfThenStateAction{
		If: cond,
	}
}
func (i *IfThenStateAction) Then(action func([]types.Action) (types.Action, bool)) StateAction {
	i.T = action
	return func(s types.State, actions []types.Action) (types.Action, bool) {
		if i.If(s) {
			return i.T(actions)
		}
		return nil, false
	}
}

type StrictPolicy struct {
	types.Policy
	conds []StateAction
}

func NewStrictPolicy(def types.Policy) *StrictPolicy {
	return &StrictPolicy{
		Policy: def,
		conds:  make([]StateAction, 0),
	}
}

var _ types.Policy = &StrictPolicy{}

func (s *StrictPolicy) AddPolicy(sa StateAction) {
	s.conds = append(s.conds, sa)
}

func (s *StrictPolicy) NextAction(step int, state types.State, actions []types.Action) (types.Action, bool) {
	for _, c := range s.conds {
		a, ok := c(state, actions)
		if ok {
			return a, ok
		}
	}
	return s.Policy.NextAction(step, state, actions)
}
