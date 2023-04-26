package policies

import "github.com/zeu5/raft-rl-test/types"

type BonusPolicyGreedy struct {
	qTable   *QTable
	horizon  int
	discount float64
	visits   *QTable
	max      bool
}

var _ types.Policy = &BonusPolicyGreedy{}

func NewBonusPolicyGreedy(horizon int, discount float64, max bool) *BonusPolicyGreedy {
	return &BonusPolicyGreedy{
		qTable:   NewQTable(),
		horizon:  horizon,
		discount: discount,
		visits:   NewQTable(),
		max:      max,
	}
}

func (b *BonusPolicyGreedy) Reset() {
	b.qTable = NewQTable()
	b.visits = NewQTable()
}

func (b *BonusPolicyGreedy) NextAction(step int, state types.State, actions []types.Action) (types.Action, bool) {
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

func (b *BonusPolicyGreedy) Update(step int, state types.State, action types.Action, nextState types.State) {
	stateHash := state.Hash()
	actionHash := action.Hash()
	nextStateHash := nextState.Hash()
	t := b.visits.Get(stateHash, actionHash, 0) + 1
	b.visits.Set(stateHash, actionHash, t)

	_, nextStateVal := b.qTable.Max(nextStateHash, 1)
	if nextStateVal > float64(b.horizon) {
		nextStateVal = float64(b.horizon)
	}
	curVal := b.qTable.Get(stateHash, actionHash, 1)

	var newVal float64
	alpha := b.alpha(step)
	if b.max {
		newVal = (1-alpha)*curVal + alpha*max(1/t, b.discount*nextStateVal)
	} else {
		newVal = (1-alpha)*curVal + alpha*(1/t+(b.discount*nextStateVal))
	}
	b.qTable.Set(stateHash, actionHash, newVal)
}

func (b *BonusPolicyGreedy) UpdateIteration(iteration int, trace *types.Trace) {

}

func (b *BonusPolicyGreedy) alpha(step int) float64 {
	return float64((b.horizon + 1) / (b.horizon + step))
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
