package policies

import "github.com/zeu5/raft-rl-test/types"

var InitState string = "Init"
var FinalState string = "Final"

type RewardMachine struct {
	predicates map[string]types.RewardFunc
	policies   map[string]types.Policy
	states     []string
}

func NewRewardMachine() *RewardMachine {
	rm := &RewardMachine{
		predicates: make(map[string]types.RewardFunc),
		policies:   make(map[string]types.Policy),
		states:     make([]string, 0),
	}
	rm.policies[FinalState] = NewBonusPolicyGreedy(50, 0.99, 0.02)
	rm.policies[InitState] = NewBonusPolicyGreedy(50, 0.99, 0.02)
	rm.states = append(rm.states, InitState)
	return rm
}

func (rm *RewardMachine) On(pred types.RewardFunc, to string) *RewardMachine {
	curState := rm.states[len(rm.states)-1]
	rm.predicates[curState] = pred
	rm.policies[curState] = NewGuidedPolicy(pred, 0.2, 0.95, 0.02)
	rm.states = append(rm.states, to)
	return rm
}

func (rm *RewardMachine) OnWithPolicy(pred types.RewardFunc, to string, policy types.Policy) *RewardMachine {
	curState := rm.states[len(rm.states)-1]
	rm.predicates[curState] = pred
	rm.predicates[curState] = pred
	rm.policies[curState] = policy
	rm.states = append(rm.states, to)
	return rm
}

func (rm *RewardMachine) WithExplorationPolicy(policy types.Policy) *RewardMachine {
	rm.policies[FinalState] = policy
	return rm
}

type RewardMachinePolicy struct {
	curRMState    string
	curRmStatePos int
	rm            *RewardMachine

	curTraceSegments []int
}

func NewRewardMachinePolicy(rm *RewardMachine) *RewardMachinePolicy {
	return &RewardMachinePolicy{
		curRMState:       InitState,
		curRmStatePos:    0,
		rm:               rm,
		curTraceSegments: make([]int, 0),
	}
}

var _ types.Policy = &RewardMachinePolicy{}

func (rp *RewardMachinePolicy) UpdateIteration(iteration int, trace *types.Trace) {
	start := 0
	for i, end := range rp.curTraceSegments {
		segment := trace.Slice(start, end)
		rmState := rp.rm.states[i]
		rmStatePolicy := rp.rm.policies[rmState]
		rmStatePolicy.UpdateIteration(iteration, segment)

		start = end
	}

	// Resetting values at the end of an iteration
	rp.curRMState = InitState
	rp.curRmStatePos = 0
	rp.curTraceSegments = make([]int, 0)
}

func (rp *RewardMachinePolicy) NextAction(step int, state types.State, actions []types.Action) (types.Action, bool) {
	curPolicy := rp.rm.policies[rp.curRMState]
	return curPolicy.NextAction(step, state, actions)
}

func (rp *RewardMachinePolicy) Update(step int, state types.State, action types.Action, nextState types.State) {
	curPolicy := rp.rm.policies[rp.curRMState]
	curPolicy.Update(step, state, action, nextState)

	// Change this - allow to skip to a later state if the predicate is true
	transitionPredicate, ok := rp.rm.predicates[rp.curRMState]
	if ok && transitionPredicate(state, nextState) {
		rp.curTraceSegments = append(rp.curTraceSegments, step)
		if rp.curRmStatePos+1 >= len(rp.rm.states) {
			// Something is wrong need to panic
			panic("reached end unexpectedly")
		}
		nextRMState := rp.rm.states[rp.curRmStatePos+1]
		rp.curRMState = nextRMState
		rp.curRmStatePos = rp.curRmStatePos + 1
	}
}

func (rp *RewardMachinePolicy) Reset() {
	for _, policy := range rp.rm.policies {
		policy.Reset()
	}
	rp.curRMState = InitState
	rp.curRmStatePos = 0
	rp.curTraceSegments = make([]int, 0)
}
