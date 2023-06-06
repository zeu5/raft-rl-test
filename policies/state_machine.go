package policies

import (
	"math/rand"
	"time"

	"github.com/zeu5/raft-rl-test/types"
)

type StateMachinePolicy struct {
	states	[]string	// list of state_names (string), giving a total order: once a predicate returns TRUE -> that state_name is set, independently from next predicates.
						// last one should be a "default" state with always TRUE predicate, the initial state of the machine
	predicates   map[string]function()	// map	state_name -> predicate, function returns a Bool: True if state satisfies the predicate
	policies_map map[string]types.Policy	// map state_name -> policy object

	active_state string	// name of the current active state
	episode_trajectory	[](string, (types.State, types.Action, types.State))	// list of (state_name, (state, action, next_state))
}

var _ types.Policy = &StateMachinePolicy{} // what is this line for?

func NewStateMachinePolicy(predicates [](pred function(), state_name string), policies [](state_name string, policy types.Policy, policy_config map?)) *StateMachinePolicy {

	policies_map = make(map[string]types.Policy)

	for _, (state_name, PolicyType, policy_config) := range policies { // instantiate the new policy with the provided configuration
		policy = NewPolicyType(policy_config)
		policies_map[state_name] = policy // add it to the map
	}

	return &StateMachinePolicy{
		predicates:	predicates,
		policies_map: policies_map,
		active_state: "default",
	}
}

func (g *StateMachinePolicy) Reset() {
	// foreach policy in the list -> call reset
}

func (g *StateMachinePolicy) UpdateIteration(iteration int, trace *types.Trace) {
	// creates sub-sequences of the trajectory based on active_state

	// call the UpdateIteration method giving the generated subsequence to the corresponding policy (state_name)
}

func (g *StateMachinePolicy) NextAction(step int, state types.State, actions []types.Action) (types.Action, bool) {
	// calls the method on the active policy
	return g.policies_map[g.active_state].NextAction(step, state, actions)
}

func (g *StateMachinePolicy) Update(step int, state types.State, action types.Action, nextState types.State) {
	
	if online_updates {	// call the method on the active policy - if we do online updates
		g.policies_map[g.active_state].Update(step, state, action, nextState)
	} else { // store the (state, action, next_state) tuple in the episode trajectory together with the active_state
		g.episode_trajectory.append((active_state,(state, action, nextState)))
	}
	

	// apply function on the ??? nextState? both current and next?
	// and eventually change the active state
	g.active_state = g.CheckActiveState(state, next_state)
}

func (g *StateMachinePolicy) CheckActiveState(state types.State, nextState types.State) { // can be based only on nextState...
	for state_name := range g.states {
		if g.predicates[state_name](next_state /* + state? */) {
			return state_name
		}
	}

	return "default" // or the predicate can be a function that returns TRUE, it is the initial state, should be at the end of the list.
}
