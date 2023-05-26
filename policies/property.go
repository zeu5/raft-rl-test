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
	rewards []types.RewardFunc
	rand    *rand.Rand
}

var _ types.Policy = &GuidedPolicy{}

func NewGuidedPolicy(rewards []types.RewardFunc, alpha, gamma, epsilon float64) *GuidedPolicy {
	return &GuidedPolicy{
		qTable:  NewQTable(),
		alpha:   alpha,
		gamma:   gamma,
		epsilon: epsilon,
		rewards: rewards,
		rand:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (g *GuidedPolicy) Reset() {
	g.qTable = NewQTable()
}

func (g *GuidedPolicy) UpdateIteration(iteration int, trace *types.Trace) {
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
	reward := 0
	for _, r := range g.rewards {
		if r(state, nextState) {
			reward = 1
			break
		}
	}
	stateHash := state.Hash()
	nextStateHash := nextState.Hash()
	actionKey := action.Hash()

	curVal := g.qTable.Get(stateHash, actionKey, 0)
	_, max := g.qTable.Max(nextStateHash, 0)

	nextVal := (1-g.alpha)*curVal + g.alpha*(float64(reward)+g.gamma*max)

	g.qTable.Set(stateHash, actionKey, nextVal)
}
