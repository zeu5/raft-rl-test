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
}

func NewBonusPolicySoftMax(horizon int, discount float64, temperature float64) *BonusPolicySoftMax {
	return &BonusPolicySoftMax{
		BonusPolicyGreedy: NewBonusPolicyGreedy(horizon, discount, 0),
		temperature:       temperature,
		rand:              rand.NewSource(uint64(time.Now().UnixNano())),
	}
}

func (b *BonusPolicySoftMax) NextAction(step int, state types.State, actions []types.Action) (types.Action, bool) {
	stateHash := state.Hash()

	sum := float64(0)
	weights := make([]float64, len(actions))
	vals := make([]float64, len(actions))

	for i, action := range actions {
		vals[i] = b.qTable.Get(stateHash, action.Hash(), 1) * (1 / b.temperature)
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
