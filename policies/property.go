package policies

import (
	"math/rand"
	"time"

	"github.com/zeu5/raft-rl-test/types"
)

type GuidedPolicy struct {
	qTable  *QTable
	alpha   float64
	gamma   float64
	epsilon float64
	reward  types.RewardFuncSingle
	rand    *rand.Rand
}

var _ RMPolicy = &GuidedPolicy{}

func NewGuidedPolicy(reward types.RewardFuncSingle, alpha, gamma, epsilon float64) *GuidedPolicy {
	return &GuidedPolicy{
		qTable:  NewQTable(),
		alpha:   alpha,
		gamma:   gamma,
		epsilon: epsilon,
		reward:  reward,
		rand:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (g *GuidedPolicy) Record(path string) {
	g.qTable.Record(path)
}

func (g *GuidedPolicy) Reset() {
	g.qTable = NewQTable()
}

func (g *GuidedPolicy) UpdateIteration(iteration int, trace *types.Trace) {
}

func (g *GuidedPolicy) UpdateIterationRm(iteration int, trace *RMTrace) {
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

func (g *GuidedPolicy) Update(sCtx *types.StepContext) {
	state := sCtx.State
	action := sCtx.Action
	nextState := sCtx.NextState

	reward := 0
	if g.reward(nextState) {
		reward = 1
	}
	stateHash := state.Hash()
	nextStateHash := nextState.Hash()
	actionKey := action.Hash()

	curVal := g.qTable.Get(stateHash, actionKey, 0)
	_, max := g.qTable.Max(nextStateHash, 0)

	nextVal := (1-g.alpha)*curVal + g.alpha*(float64(reward)+g.gamma*max)

	g.qTable.Set(stateHash, actionKey, nextVal)
}

func (g *GuidedPolicy) UpdateRm(step int, state types.State, action types.Action, nextState types.State, rwd bool, oos bool) {
	reward := 0
	if rwd {
		reward = 1
	}
	stateHash := state.Hash()
	nextStateHash := nextState.Hash()
	actionKey := action.Hash()

	curVal := g.qTable.Get(stateHash, actionKey, 0)
	_, max := g.qTable.Max(nextStateHash, 0)

	nextVal := (1-g.alpha)*curVal + g.alpha*(float64(reward)+g.gamma*max)

	g.qTable.Set(stateHash, actionKey, nextVal)
}
