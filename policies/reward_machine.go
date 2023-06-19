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

type segmentInfo struct {
	end       int
	nextState string
}

type RewardMachinePolicy struct {
	curRMState    string
	curRmStatePos int
	rm            *RewardMachine

	curTraceSegments []segmentInfo
}

func NewRewardMachinePolicy(rm *RewardMachine) *RewardMachinePolicy {
	return &RewardMachinePolicy{
		curRMState:       InitState,
		curRmStatePos:    0,
		rm:               rm,
		curTraceSegments: make([]segmentInfo, 0),
	}
}

var _ types.Policy = &RewardMachinePolicy{}

func (rp *RewardMachinePolicy) UpdateIteration(iteration int, trace *types.Trace) {
	start := 0
	policy := rp.rm.policies[InitState]
	for _, seg := range rp.curTraceSegments {
		segment := trace.Slice(start, seg.end)
		policy.UpdateIteration(iteration, segment)

		start = seg.end
		policy = rp.rm.policies[seg.nextState]
	}

	if start < trace.Len() {
		lastSlice := trace.Slice(start, trace.Len())
		explorationPolicy := rp.rm.policies[FinalState]
		explorationPolicy.UpdateIteration(iteration, lastSlice)
	}

	// Resetting values at the end of an iteration
	rp.curRMState = InitState
	rp.curRmStatePos = 0
	rp.curTraceSegments = make([]segmentInfo, 0)
}

func (rp *RewardMachinePolicy) NextAction(step int, state types.State, actions []types.Action) (types.Action, bool) {
	curPolicy := rp.rm.policies[rp.curRMState]
	return curPolicy.NextAction(step, state, actions)
}

func (rp *RewardMachinePolicy) Update(step int, state types.State, action types.Action, nextState types.State) {
	curPolicy := rp.rm.policies[rp.curRMState]
	curPolicy.Update(step, state, action, nextState)

	for i := len(rp.rm.states) - 1; i > rp.curRmStatePos; i-- {
		rmState := rp.rm.states[i]
		predicate, ok := rp.rm.predicates[rmState]
		if ok && predicate(state, nextState) {
			rp.curTraceSegments = append(rp.curTraceSegments, segmentInfo{end: step, nextState: rmState})
			rp.curRMState = rmState
			rp.curRmStatePos = i
			break
		}
	}
}

func (rp *RewardMachinePolicy) Reset() {
	for _, policy := range rp.rm.policies {
		policy.Reset()
	}
	rp.curRMState = InitState
	rp.curRmStatePos = 0
	rp.curTraceSegments = make([]segmentInfo, 0)
}
