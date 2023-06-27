package policies

import "github.com/zeu5/raft-rl-test/types"

var InitState string = "Init"
var FinalState string = "Final"

type RewardMachine struct {
	predicates map[string]types.RewardFuncSingle
	policies   map[string]types.RmPolicy
	states     []string
}

func NewRewardMachine(pred types.RewardFuncSingle) *RewardMachine {
	rm := &RewardMachine{
		predicates: make(map[string]types.RewardFuncSingle),
		policies:   make(map[string]types.RmPolicy),
		states:     make([]string, 0),
	}
	rm.policies[FinalState] = NewBonusPolicyGreedy(0.1, 0.99, 0.02)
	rm.policies[InitState] = NewBonusPolicyGreedy(0.1, 0.99, 0.02)
	rm.states = append(rm.states, InitState)
	rm.states = append(rm.states, FinalState)
	return rm
}

// add a new state in the reward machine, in the second last position (before final exploration), with predicate to go to 'to'?
func (rm *RewardMachine) On(pred types.RewardFuncSingle, to string) *RewardMachine {
	curState := rm.states[len(rm.states)-1]
	rm.predicates[curState] = pred
	rm.policies[curState] = NewGuidedPolicy(pred, 0.2, 0.95, 0.02)
	rm.states = append(rm.states, to)
	return rm
}

// add a new state in the reward machine, in the second last position (before final exploration), with predicate to go to 'to'?
func (rm *RewardMachine) OnWithPolicy(pred types.RewardFuncSingle, to string, policy types.RmPolicy) *RewardMachine {
	curState := rm.states[len(rm.states)-1]
	rm.predicates[curState] = pred
	rm.predicates[curState] = pred
	rm.policies[curState] = policy
	rm.states = append(rm.states, to)
	return rm
}

func (rm *RewardMachine) WithExplorationPolicy(policy types.RmPolicy) *RewardMachine {
	rm.policies[FinalState] = policy
	return rm
}

type segmentInfo struct {
	end       int
	nextState string
}

type transition struct {
	state     types.State
	action    types.Action
	nextState types.State
	reward    bool
}

type RewardMachinePolicy struct {
	curRMState    string
	curRmStatePos int
	rm            *RewardMachine

	curTraceSegments map[int][]transition
}

func NewRewardMachinePolicy(rm *RewardMachine) *RewardMachinePolicy {
	return &RewardMachinePolicy{
		curRMState:       InitState,
		curRmStatePos:    0,
		rm:               rm,
		curTraceSegments: make(map[int][]transition),
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
	rp.curTraceSegments = make(map[int][]transition, 0)
}

func (rp *RewardMachinePolicy) NextAction(step int, state types.State, actions []types.Action) (types.Action, bool) {

	if step == 0 {
		for i := len(rp.rm.states) - 1; i >= 0; i-- { // for all rm_states starting from the last
			rmState := rp.rm.states[i]
			predicate, ok := rp.rm.predicates[rmState]
			if ok && predicate(state) { // if current transition satisfies predicate for that rm_state
				rp.curTraceSegments = append(rp.curTraceSegments, segmentInfo{end: step, nextState: rmState})
				rp.curRMState = rmState // change current state of the rm
				rp.curRmStatePos = i    // and index
				break
			}
		}
	}

	curPolicy := rp.rm.policies[rp.curRMState]
	return curPolicy.NextAction(step, state, actions)
}

func (rp *RewardMachinePolicy) Update(step int, state types.State, action types.Action, nextState types.State) {
	curPolicy := rp.rm.policies[rp.curRMState]
	curRmStatePos := rp.curRmStatePos
	reward := false

	for i := len(rp.rm.states) - 1; i >= 0; i-- { // for all rm_states starting from the last
		rmState := rp.rm.states[i]
		predicate, ok := rp.rm.predicates[rmState]
		if ok && predicate(nextState) { // if current transition satisfies predicate for that rm_state
			rp.curTraceSegments = append(rp.curTraceSegments, segmentInfo{end: step, nextState: rmState})
			if i > curRmStatePos {
				reward = true
			}
			rp.curRMState = rmState // change current state of the rm
			rp.curRmStatePos = i    // and index
			break
		}
	}

	// NEED TO ADD INFO IF RM TRANSITIONED... THAT IS THE REWARD TRUE => 1, FALSE => 0
	curPolicy.UpdateRm(step, state, action, nextState, reward) // call the single step update function on the current policy (followed to take the step)
}

func (rp *RewardMachinePolicy) Reset() {
	for _, policy := range rp.rm.policies {
		policy.Reset()
	}
	rp.curRMState = InitState
	rp.curRmStatePos = 0
	rp.curTraceSegments = make(map[int][]transition)
}
