package policies

import (
	"math/rand"
	"time"

	"github.com/zeu5/raft-rl-test/types"
)

type GuidedPolicy struct {
	qTable   *QTable
	alpha    float64
	gamma    float64
	epsilon  float64
	property *types.Monitor
	rand     *rand.Rand
}

var _ types.Policy = &GuidedPolicy{}

func NewGuidedPolicy(monitor *types.Monitor, alpha, gamma, epsilon float64) *GuidedPolicy {
	return &GuidedPolicy{
		qTable:   NewQTable(),
		alpha:    alpha,
		gamma:    gamma,
		epsilon:  epsilon,
		property: monitor,
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (g *GuidedPolicy) Reset() {
	g.qTable = NewQTable()
}

func (g *GuidedPolicy) UpdateIteration(iteration int, trace *types.Trace) {
	prefix, ok := g.property.Check(trace)
	if ok {
		for i := prefix.Len() - 1; i > 0; i-- {
			state, action, nextState, _ := prefix.Get(i)
			// Update reward
			stateHash := state.Hash()
			nextStateHash := nextState.Hash()
			actionKey := action.Hash()

			curVal := g.qTable.Get(stateHash, actionKey, 0)
			_, max := g.qTable.Max(nextStateHash, 0)

			var nextVal float64
			if i == prefix.Len()-1 {
				// last step, give 1 reward and 0 for next state
				nextVal = (1-g.alpha)*curVal + g.alpha*(1+g.gamma*max)
			} else {
				// otherwise, update with zero reward + V(nextState)
				nextVal = (1-g.alpha)*curVal + g.alpha*(0+g.gamma*max)
			}
			g.qTable.Set(stateHash, actionKey, nextVal)
		}
	}
}

func (g *GuidedPolicy) NextAction(step int, state types.State, actions []types.Action) (types.Action, bool) {
	if g.rand.Float64() < g.epsilon {
		i := g.rand.Intn(len(actions))
		return actions[i], true
	}

	actionsMap := make(map[string]types.Action)
	actionKeys := make([]string, len(actions))
	for i, a := range actions {
		aKey := a.Hash()
		actionKeys[i] = aKey
		actionsMap[aKey] = a
	}
	permutedActionKeys := make([]string, len(actionKeys))
	for i, val := range g.rand.Perm(len(actionKeys)) {
		permutedActionKeys[i] = actionKeys[val]
	}
	maxActionKey, _ := g.qTable.MaxAmong(state.Hash(), permutedActionKeys, 0)
	if maxActionKey == "" {
		return nil, false
	}
	return actionsMap[maxActionKey], true
}

func (g *GuidedPolicy) Update(step int, state types.State, action types.Action, nextState types.State) {

}
