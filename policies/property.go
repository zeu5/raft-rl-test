package policies

import (
	"math"

	"github.com/zeu5/raft-rl-test/types"
	"gonum.org/v1/gonum/stat/sampleuv"
)

type GuidedPolicy struct {
	qTable   *QTable
	alpha    float64
	gamma    float64
	property *types.Monitor
}

var _ types.Policy = &GuidedPolicy{}

func NewGuidedPolicy(monitor *types.Monitor, alpha, gamma float64) *GuidedPolicy {
	return &GuidedPolicy{
		qTable:   NewQTable(),
		alpha:    alpha,
		gamma:    gamma,
		property: monitor,
	}
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
	} else {
		for i := trace.Len() - 1; i > 0; i-- {
			state, action, nextState, _ := trace.Get(i)
			stateHash := state.Hash()
			nextStateHash := nextState.Hash()
			actionKey := action.Hash()

			curVal := g.qTable.Get(stateHash, actionKey, 0)
			// max := 0.0
			// if g.qTable.Exists(nextStateHash) {
			_, max := g.qTable.Max(nextStateHash, 0)
			// }
			nextVal := (1-g.alpha)*curVal + g.alpha*(0+g.gamma*max)
			g.qTable.Set(stateHash, actionKey, nextVal)
		}
	}
}

func (g *GuidedPolicy) NextAction(step int, state types.State, actions []types.Action) (types.Action, bool) {
	// Softmax over actions
	stateHash := state.Hash()

	sum := float64(0)
	weights := make([]float64, len(actions))
	vals := make([]float64, len(actions))
	for i, a := range actions {
		val := g.qTable.Get(stateHash, a.Hash(), 0)
		e := math.Exp(val)
		vals[i] = e
		sum += e
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

func (g *GuidedPolicy) Update(step int, state types.State, action types.Action, nextState types.State) {

}
