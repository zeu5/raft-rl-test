package policies

import (
	"math"
	"math/rand"
	"time"

	"github.com/zeu5/raft-rl-test/rl"
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
	if _, ok := q.table[state][action]; !ok {
		q.table[state][action] = val
	}
}

func (q *QTable) HasState(state string) bool {
	_, ok := q.table[state]
	return ok
}

func (q *QTable) Max(state string, def float64) (string, float64) {
	if _, ok := q.table[state]; !ok {
		q.table[state] = make(map[string]float64)
		return "", def
	}
	maxAction := ""
	maxVal := float64(math.MinInt)
	for a, val := range q.table[state] {
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
	properties  []*rl.Monitor
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
	// just an index for which property is being followed, 0 => no property
	currentProp int
	rand        *rand.Rand
}

var _ rl.Policy = &PropertyGuidedPolicy{}

func NewPropertyGuidedPolicy(properties []*rl.Monitor, alpha, gamma, epsilon float64) *PropertyGuidedPolicy {
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
		currentProp: -1,
		rand:        rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	for i := range properties {
		policy.propQTables[i] = NewQTable()
	}
	return policy
}

// Run at the end of each episode, use the trace to update policies and choose next property
func (p *PropertyGuidedPolicy) UpdateIteration(iteration int, trace *rl.Trace) {
	// check and update policies for all properties
	for i := 0; i < len(p.properties); i++ {
		p.updatePolicy(i, trace)
	}

	// choose prop to follow in next episode
	p.chooseProperty(iteration)

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
func (p *PropertyGuidedPolicy) updatePolicy(propertyIndex int, trace *rl.Trace) {
	prop := p.properties[propertyIndex]
	// last (s,a,s') should be where s' fulfills the property.
	prefix, ok := prop.Check(trace)
	if !ok {
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

func (p *PropertyGuidedPolicy) NextAction(step int, state rl.State, actions []rl.Action) (rl.Action, bool) {
	stateHash := state.Hash()

	if p.currentProp != -1 {
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
		actionMap := make(map[string]rl.Action)
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
func (p *PropertyGuidedPolicy) nextActionExp(statehash string, actions []rl.Action) (rl.Action, bool) {

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
func (p *PropertyGuidedPolicy) Update(step int, state rl.State, action rl.Action, nextState rl.State) {
	stateHash := state.Hash()

	nextStateHash := nextState.Hash()
	actionKey := action.Hash()
	curVal := p.expQTable.Get(stateHash, actionKey, 0)
	// take V(nextState)
	_, max := p.expQTable.Max(nextStateHash, 0)
	// update QVal
	nextVal := (1-p.alpha)*curVal + p.alpha*(-1+p.gamma*max)
	p.expQTable.Set(stateHash, actionKey, nextVal)
}
