package policies

import (
	"time"

	"github.com/zeu5/raft-rl-test/types"
	"golang.org/x/exp/rand"
)

type BonusPolicyGreedy struct {
	qTable   *QTable
	alpha    float64
	discount float64
	visits   *QTable
	epsilon  float64
	rand     rand.Rand

	max bool
}

var _ RMPolicy = &BonusPolicyGreedy{}

func NewBonusPolicyGreedy(alpha, discount, epsilon float64, max bool) *BonusPolicyGreedy {
	return &BonusPolicyGreedy{
		qTable:   NewQTable(),
		alpha:    alpha,
		discount: discount,
		visits:   NewQTable(),
		epsilon:  epsilon,
		rand:     *rand.New(rand.NewSource(uint64(time.Now().UnixNano()))),
		max:      max,
	}
}

func (b *BonusPolicyGreedy) Record(path string) {
	b.qTable.Record(path)
}

func (b *BonusPolicyGreedy) Reset() {
	b.qTable = NewQTable()
	b.visits = NewQTable()
}

func (b *BonusPolicyGreedy) NextAction(step int, state types.State, actions []types.Action) (types.Action, bool) {

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

func (b *BonusPolicyGreedy) Update(sCtx *types.StepContext) {

}

func (b *BonusPolicyGreedy) UpdateInternal(state types.State, action types.Action, nextState types.State, outOfSpace bool) {
	stateHash := state.Hash()
	actionHash := action.Hash()
	nextStateHash := nextState.Hash()
	t := b.visits.Get(stateHash, actionHash, 0) + 1
	b.visits.Set(stateHash, actionHash, t)

	nextStateVal := 0.0
	// if not horizon reached, get the value of the next state
	if !outOfSpace {
		_, nextStateVal = b.qTable.Max(nextStateHash, 1)
	}
	curVal := b.qTable.Get(stateHash, actionHash, 1)

	newVal := 0.0

	if b.max {
		newVal = (1-b.alpha)*curVal + b.alpha*max(1/t, b.discount*nextStateVal)
	} else {
		newVal = (1-b.alpha)*curVal + b.alpha*(1/t+b.discount*nextStateVal)
	}

	b.qTable.Set(stateHash, actionHash, newVal)
}

func (b *BonusPolicyGreedy) UpdateRm(step int, state types.State, action types.Action, nextState types.State, rwd bool, oos bool, reachedFinal bool, reachedFinalStep int) {
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

func (b *BonusPolicyGreedy) UpdateIteration(iteration int, trace *types.Trace) {
	lastIndex := trace.Len() - 1

	for i := lastIndex; i > -1; i-- { // going backwards in the segment
		outOfSpace := false
		if i == lastIndex {
			outOfSpace = true
		}
		state, action, nextState, ok := trace.Get(i)
		if ok {
			b.UpdateInternal(state, action, nextState, outOfSpace)
		}
	}

}

func (b *BonusPolicyGreedy) UpdateIterationRm(iteration int, trace *RMTrace, reachedFinal bool, reachedFinalStep int) {
	lastIndex := trace.Len() - 1

	for i := lastIndex; i > -1; i-- { // going backwards in the segment
		step, state, action, nextState, reward, outOfSpace, ok := trace.Get(i)
		if ok {
			b.UpdateRm(step, state, action, nextState, reward, outOfSpace, reachedFinal, reachedFinalStep)
		}
	}

}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
