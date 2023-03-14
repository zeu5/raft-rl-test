package policies

import (
	"math"

	"github.com/zeu5/raft-rl-test/types"
	"gonum.org/v1/gonum/stat/sampleuv"
)

type PropertyGuidedPolicy struct {
	properties []*types.Monitor
	QTableExp   map[string]map[string]float64
	QTablesProp	*map[string]map[string]float64
	alpha    float64
	gamma    float64
	epsilon	float64 // probability to still follow general Exploration instead of prop, greedy over QTablesProp[prop] otherwise

	episodes int // keep track of total number of episodes 
	prop_episodes []int // keep track of number of episodes each property has been followed
	current_prop int // just an index for which property is being followed, 0 => no property
}

var _ types.Policy = &PropertyGuidedPolicy{}

func NewPropertyGuidedPolicy(properties []*types.Monitor, alpha, gamma, epsilon float64) *PropertyGuidedPolicy {
	return &PropertyGuidedPolicy{
		properties: []properties,
		QTableExp:   make(map[string]map[string]float64), // general exploration table, update it all the times, use it when not following any prop or state is unknown for the prop
		QTablesProp: []make(map[string]map[string]float64), // list of QTables, one for each prop, updated only for trajectories that fulfill the prop
		alpha:    alpha,
		gamma:    gamma,
		epsilon:	epsilon,

		episodes := 0,
		current_prop := -1, // start with no property
	}
}

// Run at the end of each episode, use the trace to update policies and choose next property
func (p *PropertyGuidedPolicy) UpdateIteration(iteration int, trace *types.Trace) {
	for i := 0; i < properties.Len(); i++ { // check and update policies for all properties
		p.updatePolicy(properties[i], trace)
	}

	p.choose_property(iteration) // choose prop to follow in next episode
}

// Choose which property to use next, according to some policy
func (p *PropertyGuidedPolicy) choose_property(iteration int) {
	if p.current_prop == (p.properties.Len() - 1) {
		p.current_prop = -1
	} else {
		p.current_prop += 1
	}
}

// Takes a property and a trace and updates the Qvalues backwards giving reward 1 for the
// last (state,action) pair.
func (p *PropertyGuidedPolicy) updatePolicy(prop property, trace *types.Trace) {
	prefix, ok := prop.Check(trace) // last (s,a,s') should be where s' fulfills the property.
	if !ok {
		return
	}

	for i := prefix.Len() - 1; i >= 0; i-- {
		state, action, nextState, _ := prefix.Get(i)
		stateHash := state.Hash()

		nextStateHash := nextState.Hash()
		actionKey := action.Hash()
		if _, ok := p.QTable[stateHash]; !ok {
			return
		}
		if _, ok := p.QTable[stateHash][actionKey]; !ok {
			return
		}
		curVal := p.QTable[stateHash][actionKey]
		max := float64(0)
		if _, ok := p.QTable[nextStateHash]; ok {
			for _, val := range p.QTable[nextStateHash] {
				if val > max {
					max = val
				}
			}
		}

		if i == prefix.Len() - 1 { // last step, give 1 reward and 0 for next state
			nextVal := (1-p.alpha)*curVal + p.alpha*(p.gamma*1)
		} else { // otherwise, update with zero reward + V(nextState)
			nextVal := (1-p.alpha)*curVal + p.alpha*(p.gamma*max)
		}
		
		p.QTable[stateHash][actionKey] = nextVal
	}
}

func (p *PropertyGuidedPolicy) NextAction(step int, state types.State, actions []types.Action) (types.Action, bool) {
	stateHash := state.Hash()

	if p.current_prop != -1 { // there is a selected prop policy
		if _, ok := p.QTablesProp[p.current_prop][stateHash]; !ok { // state unkown for the prop policy, just follow exploration
			return p.nextActionExp(stateHash, actions)
		}
		// ... TO DO ...
		// sample random number [0,1]: if < epsilon:
		//   use general Exploration
		// else:
		//   choose action greedily over QTablesProp[p.current_prop][stateHash][action]

		for _, a := range actions {
			aName := a.Hash()
			if _, ok := p.QTable[stateHash][aName]; !ok {
				p.QTable[stateHash][aName] = 0
			}
		}
	} else { // no prop selected, use general Exploration
		return p.nextActionExp(stateHash, actions)
	}

	if _, ok := p.QTablesProp[p.current_prop][stateHash]; !ok { // state unkown for the prop policy, just follow exploration
		p.QTable[stateHash] = make(map[string]float64)
	}
	

	for _, a := range actions {
		aName := a.Hash()
		if _, ok := p.QTable[stateHash][aName]; !ok {
			p.QTable[stateHash][aName] = 0
		}
	}

	sum := float64(0)
	weights := make([]float64, len(actions))
	vals := make([]float64, len(actions))

	for i, action := range actions {
		val := p.QTable[stateHash][action.Hash()]
		exp := math.Exp(val)
		vals[i] = exp
		sum += exp
	}

	for i, v := range vals {
		weights[i] = v / sum
	}
	i, ok := sampleuv.NewWeighted(weights, nil).Take()
	if !ok {
		return nil, false
	}
	return actions[i], true
}

// choose next action based on the general Exploration policy
func (p *PropertyGuidedPolicy) nextActionExp(statehash string, actions []types.Action) (types.Action, bool) {
	
	if _, ok := p.QTableExp[stateHash]; !ok { // if never seen, create new map for the state
		p.QTableExp[stateHash] = make(map[string]float64)
	}
	

	for _, a := range actions { // initialize non-existant QVals to 0
		aName := a.Hash()
		if _, ok := p.QTableExp[stateHash][aName]; !ok {
			p.QTableExp[stateHash][aName] = 0
		}
	}

	// compute softmax and sample action according to the distribution
	sum := float64(0)
	weights := make([]float64, len(actions))
	vals := make([]float64, len(actions))

	for i, action := range actions { // better to normalize everything subtracting Max(vals) to vals
		val := p.QTableExp[stateHash][action.Hash()]
		exp := math.Exp(val)
		vals[i] = exp
		sum += exp
	}

	for i, v := range vals {
		weights[i] = v / sum
	}
	i, ok := sampleuv.NewWeighted(weights, nil).Take()
	if !ok {
		return nil, false
	}
	return actions[i], true
}

// update the general Exploration policy, ??? it can be done at each step, or at the end of each episode ???
func (p *PropertyGuidedPolicy) Update(step int, state State, action Action, nextState State) {
	stateHash := state.Hash()

	nextStateHash := nextState.Hash()
	actionKey := action.Hash()
	if _, ok := p.QTableExp[stateHash]; !ok { // if never seen, create new map for the state
		p.QTableExp[stateHash] = make(map[string]float64)
	}
	if _, ok := p.QTableExp[stateHash][actionKey]; !ok { // initialize non-existant QVals to 0
		p.QTableExp[stateHash][actionKey] = 0
	}
	curVal := p.QTableExp[stateHash][actionKey]
	// take V(nextState)
	max := float64(0)
	if _, ok := p.QTableExp[nextStateHash]; ok {
		for _, val := range p.QTableExp[nextStateHash] {
			if val > max {
				max = val
			}
		}
	}
	// update QVal
	nextVal := (1-p.alpha)*curVal + p.alpha*(-1+s.gamma*max)
	p.QTableExp[stateHash][actionKey] = nextVal
}
