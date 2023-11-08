package policies

import (
	"math"
	"math/rand"
	"time"

	"github.com/zeu5/raft-rl-test/types"
	"gonum.org/v1/gonum/stat/sampleuv"
)

type QTable struct {
	table map[string]map[string]float64
}

func NewQTable() *QTable {
	return &QTable{
		table: make(map[string]map[string]float64),
	}
}

func (q *QTable) Get(state, action string, def float64) float64 {
	if _, ok := q.table[state]; !ok {
		q.table[state] = make(map[string]float64)
	}
	if _, ok := q.table[state][action]; !ok {
		q.table[state][action] = def
	}
	return q.table[state][action]
}

func (q *QTable) Set(state, action string, val float64) {
	if _, ok := q.table[state]; !ok {
		q.table[state] = make(map[string]float64)
	}
	q.table[state][action] = val
}

func (q *QTable) HasState(state string) bool {
	_, ok := q.table[state]
	return ok
}

func (q *QTable) Max(state string, def float64) (string, float64) {
	if _, ok := q.table[state]; !ok { // if state is not in the QTable at all, return default value
		q.table[state] = make(map[string]float64)
		return "", def
	}
	maxAction := ""
	maxVal := float64(math.MinInt)
	for a, val := range q.table[state] { // for all the actions
		if val > maxVal {
			maxAction = a
			maxVal = val
		}
	}
	if maxAction == "" {
		return "", def
	}
	return maxAction, maxVal
}

func (q *QTable) Exists(state string) bool {
	_, ok := q.table[state]
	return ok
}

func (q *QTable) MaxAmong(state string, actions []string, def float64) (string, float64) {
	if _, ok := q.table[state]; !ok {
		q.table[state] = make(map[string]float64)
	}
	maxAction := ""
	maxVal := float64(math.MinInt)
	for _, a := range actions {
		if _, ok := q.table[state][a]; !ok {
			q.table[state][a] = def
		}
		val := q.table[state][a]
		if val > maxVal {
			maxAction = a
			maxVal = val
		}
	}
	return maxAction, maxVal
}

type PropertyGuidedPolicy struct {
	properties  []*types.Monitor
	expQTable   *QTable
	propQTables []*QTable
	alpha       float64
	gamma       float64
	// probability to still follow general Exploration instead of prop, greedy over QTablesProp[prop] otherwise
	epsilon float64

	// keep track of total number of episodes
	episodes int
	// keep track of number of episodes each property has been followed
	propEpisodes []int
	// just an index for which property is being followed, -1 => no property
	currentProp int
	rand        *rand.Rand
	// bool to switch off the specific policy once the waypoint is reached
	reached       bool
	current_trace *types.Trace
}

var _ types.Policy = &PropertyGuidedPolicy{}

func NewPropertyGuidedPolicy(properties []*types.Monitor, alpha, gamma, epsilon float64) *PropertyGuidedPolicy {
	policy := &PropertyGuidedPolicy{
		properties: properties,
		// general exploration table, update it all the times, use it when not following any prop or state is unknown for the prop
		expQTable: NewQTable(),
		// list of QTables, one for each prop, updated only for trajectories that fulfill the prop
		propQTables:  make([]*QTable, len(properties)),
		alpha:        alpha,
		gamma:        gamma,
		epsilon:      epsilon,
		propEpisodes: make([]int, len(properties)),
		// start with no property
		currentProp:   -1,
		rand:          rand.New(rand.NewSource(time.Now().UnixNano())),
		reached:       false,
		current_trace: types.NewTrace(),
	}
	for i := range properties {
		policy.propQTables[i] = NewQTable()
	}
	return policy
}

func (p *PropertyGuidedPolicy) Reset() {
	p.expQTable = NewQTable()
	properties := len(p.propQTables)
	p.propQTables = make([]*QTable, properties)
	p.propEpisodes = make([]int, properties)
}

// Run at the end of each episode, use the trace to update policies and choose next property
func (p *PropertyGuidedPolicy) UpdateIteration(iteration int, trace *types.Trace) {
	// check and update policies for all properties
	for i := 0; i < len(p.properties); i++ {
		p.updatePolicy(i, trace)
	}

	// choose prop to follow in next episode
	p.chooseProperty(iteration)

	// restart reached and current trace for the upcoming episode
	p.reached = false
	p.current_trace = types.NewTrace()

	p.episodes += 1
}

// Choose which property to use next, according to some policy
func (p *PropertyGuidedPolicy) chooseProperty(iteration int) {
	if p.currentProp == (len(p.properties) - 1) {
		p.currentProp = -1
	} else {
		p.currentProp += 1
	}
}

// Takes a property and a trace and updates the Qvalues backwards giving reward 1 for the
// last (state,action) pair.
func (p *PropertyGuidedPolicy) updatePolicy(propertyIndex int, trace *types.Trace) {
	prop := p.properties[propertyIndex]
	// last (s,a,s') should be where s' fulfills the property.
	prefix, ok := prop.Check(trace)
	if !ok { // update the policy without reward
		p.updatePolicyUnsat(propertyIndex, trace)
		return
	}

	p.propEpisodes[propertyIndex] += 1
	propQTable := p.propQTables[propertyIndex]
	for i := prefix.Len() - 1; i >= 0; i-- {
		state, action, nextState, _ := prefix.Get(i)
		stateHash := state.Hash()

		nextStateHash := nextState.Hash()
		actionKey := action.Hash()
		curVal := propQTable.Get(stateHash, actionKey, 0)
		_, max := propQTable.Max(nextStateHash, 0)

		var nextVal float64
		if i == prefix.Len()-1 {
			// last step, give 1 reward and 0 for next state
			nextVal = (1-p.alpha)*curVal + p.alpha*(1+p.gamma*max)
		} else {
			// otherwise, update with zero reward + V(nextState)
			nextVal = (1-p.alpha)*curVal + p.alpha*(0+p.gamma*max)
		}

		propQTable.Set(stateHash, actionKey, nextVal)
	}
}

// Takes a property and a trace which does not satisfies the property, and updates the QValues
// without adding any new state to the propQTable:
//
//	Evaluates as 0 all the states that are not in the table and does not updates them.
func (p *PropertyGuidedPolicy) updatePolicyUnsat(propertyIndex int, trace *types.Trace) {

	p.propEpisodes[propertyIndex] += 1
	propQTable := p.propQTables[propertyIndex]
	for i := trace.Len() - 1; i >= 0; i-- { // full trace
		state, action, nextState, _ := trace.Get(i)
		stateHash := state.Hash()
		actionKey := action.Hash()

		if _, ok := propQTable.table[stateHash][actionKey]; ok { // (s,a) is in the propQTable
			curVal := propQTable.Get(stateHash, actionKey, 0)
			max := 0.0 // Use zero as Vval if the nextState is not in the propQTable

			nextStateHash := nextState.Hash()
			if propQTable.Exists(nextStateHash) { // nextState is in the propQTable
				_, max = propQTable.Max(nextStateHash, 0) // use Vval(nextState) for the update
			}

			nextVal := (1-p.alpha)*curVal + p.alpha*(0+p.gamma*max)

			propQTable.Set(stateHash, actionKey, nextVal)
		}

	}
}

func (p *PropertyGuidedPolicy) NextAction(step int, state types.State, actions []types.Action) (types.Action, bool) {
	stateHash := state.Hash()

	if p.currentProp != -1 && !p.reached {
		propQTable := p.propQTables[p.currentProp]
		// there is a selected prop policy
		if !propQTable.HasState(stateHash) {
			// state unkown for the prop policy, just follow exploration
			return p.nextActionExp(stateHash, actions)
		}

		// ... TO DO ...
		// sample random number [0,1]: if < epsilon:
		//   use general Exploration
		// else:
		//   choose action greedily over QTablesProp[p.current_prop][stateHash][action]
		if rand.Float64() < p.epsilon {
			return p.nextActionExp(stateHash, actions)
		}
		actionMap := make(map[string]types.Action)
		actionKeys := make([]string, len(actions))
		for i, a := range actions {
			actionKeys[i] = a.Hash()
			actionMap[a.Hash()] = a
		}
		maxAction, _ := propQTable.MaxAmong(stateHash, actionKeys, 0)
		if maxAction == "" {
			return nil, false
		}
		return actionMap[maxAction], true
	}
	// no prop selected, use general Exploration
	return p.nextActionExp(stateHash, actions)
}

// choose next action based on the general Exploration policy
func (p *PropertyGuidedPolicy) nextActionExp(statehash string, actions []types.Action) (types.Action, bool) {

	// compute softmax and sample action according to the distribution
	sum := float64(0)
	weights := make([]float64, len(actions))
	vals := make([]float64, len(actions))

	for i, action := range actions {
		// better to normalize everything subtracting Max(vals) to vals
		val := p.expQTable.Get(statehash, action.Hash(), 0)
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
func (p *PropertyGuidedPolicy) Update(step int, state types.State, action types.Action, nextState types.State) {
	stateHash := state.Hash()

	nextStateHash := nextState.Hash()
	actionKey := action.Hash()
	curVal := p.expQTable.Get(stateHash, actionKey, 0)
	// take V(nextState)
	_, max := p.expQTable.Max(nextStateHash, 0)
	// update QVal
	nextVal := (1-p.alpha)*curVal + p.alpha*(-1+p.gamma*max)
	p.expQTable.Set(stateHash, actionKey, nextVal)

	// check if prop has been reached, if so => shut down the specific policy
	if p.currentProp != -1 && !p.reached { // if not no_property and not already reached
		p.checkReached(step, state, action, nextState)
	}
}

// Appends the current step (s,a,s') to the current trace and checks whether it satisfies or not the current prop,
// if yes => sets reached to true, switching off the specific policy for the remaining steps of the episode
func (p *PropertyGuidedPolicy) checkReached(step int, state types.State, action types.Action, nextState types.State) {
	p.current_trace.Append(step, state, action, nextState)
	prop := p.properties[p.currentProp]

	if _, ok := prop.Check(p.current_trace); ok {
		p.reached = true
	}
}
