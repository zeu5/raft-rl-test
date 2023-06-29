package policies

import (
	"time"

	"math/rand"

	"github.com/zeu5/raft-rl-test/types"
)

type BonusPolicyGreedyReward struct {
	qTable   *QTable
	alpha    float64
	discount float64
	visits   *QTable
	epsilon  float64
	rand     *rand.Rand
}

var _ types.RmPolicy = &BonusPolicyGreedyReward{}

func NewBonusPolicyGreedyReward(alpha, discount, epsilon float64) *BonusPolicyGreedyReward {
	return &BonusPolicyGreedyReward{
		qTable:   NewQTable(),
		alpha:    alpha,
		discount: discount,
		visits:   NewQTable(),
		epsilon:  epsilon,
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (b *BonusPolicyGreedyReward) Reset() {
	b.qTable = NewQTable()
	b.visits = NewQTable()
}

func (b *BonusPolicyGreedyReward) NextAction(step int, state types.State, actions []types.Action) (types.Action, bool) {

	if b.rand.Float64() < b.epsilon {
		i := b.rand.Intn(len(actions))
		return actions[i], true
	}

	actionsMap := make(map[string]types.Action)
	availableActions := make([]string, len(actions))
	for i, a := range actions {
		aHash := a.Hash()
		actionsMap[aHash] = a
		availableActions[i] = aHash
	}
	maxAction, _ := b.qTable.MaxAmong(state.Hash(), availableActions, 1)
	if maxAction == "" {
		return nil, false
	}
	return actionsMap[maxAction], true
}

func (b *BonusPolicyGreedyReward) Update(step int, state types.State, action types.Action, nextState types.State) {
	stateHash := state.Hash()
	actionHash := action.Hash()
	nextStateHash := nextState.Hash()
	t := b.visits.Get(stateHash, actionHash, 0) + 1
	b.visits.Set(stateHash, actionHash, t)

	_, nextStateVal := b.qTable.Max(nextStateHash, 1)
	curVal := b.qTable.Get(stateHash, actionHash, 1)

	newVal := (1-b.alpha)*curVal + b.alpha*max(1/t, b.discount*nextStateVal)
	b.qTable.Set(stateHash, actionHash, newVal)
}

func (b *BonusPolicyGreedyReward) UpdateRm(step int, state types.State, action types.Action, nextState types.State, rwd bool) {
	stateHash := state.Hash()
	actionHash := action.Hash()
	nextStateHash := nextState.Hash()
	var r float64

	t := b.visits.Get(stateHash, actionHash, 0) + 1
	b.visits.Set(stateHash, actionHash, t)

	if rwd { // if reward, give 2 (value higher than any bonus)
		r = 2
	} else { // assign reward according to visits
		r = 1 / t
	}

	_, nextStateVal := b.qTable.Max(nextStateHash, 1)
	curVal := b.qTable.Get(stateHash, actionHash, 1)

	newVal := (1-b.alpha)*curVal + b.alpha*max(r, b.discount*nextStateVal)
	b.qTable.Set(stateHash, actionHash, newVal)
}

func (b *BonusPolicyGreedyReward) UpdateIteration(iteration int, trace *types.Trace) {

}

// func max(a, b float64) float64 {
// 	if a > b {
// 		return a
// 	}
// 	return b
// }
