package policies

import (
	"github.com/zeu5/raft-rl-test/types"
)

var InitState string = "Init"
var FinalState string = "Final"

type RewardMachine struct {
	predicates map[string]types.RewardFuncSingle
	policies   map[string]types.RmPolicy
	states     []string
}

func TruePred() types.RewardFuncSingle {
	return func(s types.State) bool {
		return true
	}
}

func NewRewardMachine(pred types.RewardFuncSingle) *RewardMachine {
	rm := &RewardMachine{
		predicates: make(map[string]types.RewardFuncSingle),
		policies:   make(map[string]types.RmPolicy),
		states:     make([]string, 0),
	}
	rm.policies[FinalState] = NewBonusPolicyGreedy(0.1, 0.99, 0.05)
	rm.policies[InitState] = NewBonusPolicyGreedyReward(0.2, 0.9, 0.05)
	rm.states = append(rm.states, InitState)
	rm.states = append(rm.states, FinalState)

	rm.predicates[InitState] = TruePred()
	rm.predicates[FinalState] = pred

	return rm
}

// TODO: change states to not point to next states
// add a new state in the reward machine, in the second last position (before final exploration), with predicate to go to 'to'?
func (rm *RewardMachine) AddState(pred types.RewardFuncSingle, name string) *RewardMachine {
	index := len(rm.states) - 1
	rm.states = append(rm.states, rm.states[index]) // duplicate last element

	// modify second-last element, insert the new state
	rm.states[index] = name
	rm.predicates[name] = pred
	rm.policies[name] = NewBonusPolicyGreedyReward(0.2, 0.9, 0.05)

	return rm
}

// add a new state in the reward machine, in the second last position (before final exploration), with predicate to go to 'to'?
func (rm *RewardMachine) AddStateWithPolicy(pred types.RewardFuncSingle, name string, policy types.RmPolicy) *RewardMachine {
	index := len(rm.states) - 1
	rm.states = append(rm.states, rm.states[index]) // duplicate last element

	// modify second-last element, insert the new state
	rm.states[index] = name
	rm.predicates[name] = pred
	rm.policies[name] = policy

	return rm
}

func (rm *RewardMachine) WithExplorationPolicy(policy types.RmPolicy) *RewardMachine {
	rm.policies[FinalState] = policy
	return rm
}

func (rm *RewardMachine) GetFinalPredicate() types.RewardFuncSingle {
	return rm.predicates[FinalState]
}

type RewardMachinePolicy struct {
	curRMState    string
	curRmStatePos int
	rm            *RewardMachine

	oneTime bool // if true, the last predicate needs to be satisfied only once throughout the episode
	reached bool // flag for the last predicate to be reached within an episode

	curTraceSegments map[string]*types.RmTrace
}

func NewRewardMachinePolicy(rm *RewardMachine, oneTime bool) *RewardMachinePolicy {
	return &RewardMachinePolicy{
		curRMState:       InitState,
		curRmStatePos:    0,
		rm:               rm,
		curTraceSegments: make(map[string]*types.RmTrace),
		oneTime:          oneTime,
		reached:          false,
	}
}

var _ types.Policy = &RewardMachinePolicy{}

// calls the update iterations methods in the policies with their respective segments
func (rp *RewardMachinePolicy) UpdateIteration(iteration int, trace *types.Trace) {
	for state, segment := range rp.curTraceSegments {
		policy := rp.rm.policies[state]
		policy.UpdateIterationRm(iteration, segment)
	}

	// Resetting values at the end of an iteration
	rp.curRMState = InitState
	rp.curRmStatePos = 0
	rp.curTraceSegments = make(map[string]*types.RmTrace)
	rp.reached = false
}

func (rp *RewardMachinePolicy) NextAction(step int, state types.State, actions []types.Action) (types.Action, bool) {

	if step == 0 {
		for i := len(rp.rm.states) - 1; i >= 0; i-- { // for all rm_states starting from the last
			rmState := rp.rm.states[i]
			predicate, ok := rp.rm.predicates[rmState]
			if ok && predicate(state) { // if current transition satisfies predicate for that rm_state
				rp.curRMState = rmState // change current state of the rm
				rp.curRmStatePos = i    // and index
				break
			}
		}
	}

	curPolicy := rp.rm.policies[rp.curRMState]
	return curPolicy.NextAction(step, state, actions)
}

// updates the trace segments for the policies to update at the end of the episode
func (rp *RewardMachinePolicy) Update(step int, state types.State, action types.Action, nextState types.State) {
	// curPolicy := rp.rm.policies[rp.curRMState]
	curRmStatePos := rp.curRmStatePos
	curRmState := rp.curRMState
	reward := false
	out_of_space := false

	newRmPosition := len(rp.rm.states) - 1
	newRmState := rp.rm.states[newRmPosition]

	if !rp.reached {
		for i := len(rp.rm.states) - 1; i >= 0; i-- { // for all rm_states starting from the last
			rmState := rp.rm.states[i]
			predicate, ok := rp.rm.predicates[rmState]
			if ok && predicate(nextState) { // if current transition satisfies predicate for that rm_state
				if i > curRmStatePos { // progressed in the machine
					reward = true
				}

				if i != rp.curRmStatePos { // changed the state
					out_of_space = true
				}

				newRmPosition = i
				newRmState = rp.rm.states[i]

				if rp.oneTime && newRmPosition == len(rp.rm.states)-1 { // if it is a oneTime machine and it reached the last state
					rp.reached = true // set the flag on
				}

				break
			}
		}
	}

	if _, ok := rp.curTraceSegments[curRmState]; !ok {
		rp.curTraceSegments[curRmState] = types.NewRmTrace()
	}
	rp.curTraceSegments[curRmState].Append(step, state, action, nextState, reward, out_of_space)

	rp.curRMState = newRmState       // change current state of the rm
	rp.curRmStatePos = newRmPosition // and index

	// NEED TO ADD INFO IF RM TRANSITIONED... THAT IS THE REWARD TRUE => 1, FALSE => 0
	// curPolicy.UpdateRm(step, state, action, nextState, reward, out_of_space) // call the single step update function on the current policy (followed to take the step)
}

func (rp *RewardMachinePolicy) Reset() {
	for _, policy := range rp.rm.policies {
		policy.Reset()
	}
	rp.curRMState = InitState
	rp.curRmStatePos = 0
	rp.curTraceSegments = make(map[string]*types.RmTrace)
	rp.reached = false
}
