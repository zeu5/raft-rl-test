package policies

import (
	"math"

	"github.com/zeu5/raft-rl-test/types"
	"gonum.org/v1/gonum/stat/sampleuv"
)

type PropertyGuidedPolicy struct {
	property *types.Monitor
	QTable   map[string]map[string]float64
	alpha    float64
	gamma    float64
}

var _ types.Policy = &PropertyGuidedPolicy{}

func NewPropertyGuidedPolicy(property *types.Monitor, alpha, gamma float64) *PropertyGuidedPolicy {
	return &PropertyGuidedPolicy{
		property: property,
		QTable:   make(map[string]map[string]float64),
		alpha:    alpha,
		gamma:    gamma,
	}
}

func (p *PropertyGuidedPolicy) UpdateIteration(iteration int, trace *types.Trace) {
	prefix, ok := p.property.Check(trace)
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
		nextVal := (1-p.alpha)*curVal + p.alpha*(1+p.gamma*max)
		p.QTable[stateHash][actionKey] = nextVal
	}
}

func (p *PropertyGuidedPolicy) NextAction(step int, state types.State, actions []types.Action) (types.Action, bool) {
	stateHash := state.Hash()

	if _, ok := p.QTable[stateHash]; !ok {
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

func (p *PropertyGuidedPolicy) Update(_ int, _ types.State, _ types.Action, _ types.State) {}
