package policies

import (
	"math"
	"time"

	"github.com/zeu5/raft-rl-test/types"
	"golang.org/x/exp/rand"
	"gonum.org/v1/gonum/stat/sampleuv"
)

type BonusPolicySoftMax struct {
	*BonusPolicyGreedy
	temperature float64
	rand        rand.Source
	normalize   bool
}

func NewBonusPolicySoftMax(alpha, discount float64, temperature float64, normalize bool) *BonusPolicySoftMax {
	return &BonusPolicySoftMax{
		BonusPolicyGreedy: NewBonusPolicyGreedy(alpha, discount, 0),
		temperature:       temperature,
		rand:              rand.NewSource(uint64(time.Now().UnixNano())),
		normalize:         normalize,
	}
}

func (b *BonusPolicySoftMax) NextAction(step int, state types.State, actions []types.Action) (types.Action, bool) {
	stateHash := state.Hash()

	sum := float64(0)
	weights := make([]float64, len(actions))
	vals := make([]float64, len(actions))
	for i, action := range actions {
		vals[i] = b.qTable.Get(stateHash, action.Hash(), 1)
	}
	if b.normalize {
		minVal := float64(math.MaxInt)
		for _, val := range vals {
			if val < minVal {
				minVal = val
			}
		}

		maxNewVal := float64(math.MinInt)
		newVals := make([]float64, len(vals))
		for i, val := range vals {
			newVals[i] = val / minVal
			if newVals[i] > maxNewVal {
				maxNewVal = newVals[i]
			}
		}
		for i, val := range newVals {
			vals[i] = val - maxNewVal
		}
	}
	// TODO: Normalize
	for i, val := range vals {
		exp := math.Exp(val)
		vals[i] = exp
		sum += exp
	}
	for i, v := range vals {
		weights[i] = v / sum
	}
	i, ok := sampleuv.NewWeighted(weights, b.rand).Take()
	if !ok {
		return nil, false
	}
	return actions[i], true
}
