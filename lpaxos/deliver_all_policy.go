package lpaxos

import (
	"math/rand"
	"time"

	"github.com/zeu5/raft-rl-test/types"
)

type OnlyDeliverPolicy struct {
	rand  *rand.Rand
	First bool
}

var _ types.Policy = &OnlyDeliverPolicy{}

func NewOnlyDeliverPolicy(first bool) *OnlyDeliverPolicy {
	return &OnlyDeliverPolicy{
		rand:  rand.New(rand.NewSource(time.Now().UnixNano())),
		First: first,
	}
}

func (r *OnlyDeliverPolicy) Record(path string) {}

func (r *OnlyDeliverPolicy) Reset() {}

func (r *OnlyDeliverPolicy) UpdateIteration(_ int, _ *types.Trace) {}

func (r *OnlyDeliverPolicy) NextAction(step int, state types.State, actions []types.Action) (types.Action, bool) {
	deliverActions := make([]types.Action, 0)
	for _, a := range actions {
		lPaxosAction, ok := a.(*LPaxosAction)
		if !ok {
			continue
		}
		if lPaxosAction.Type == "Deliver" {
			deliverActions = append(deliverActions, a)
		}
	}
	if len(deliverActions) == 0 {
		return nil, false
	}
	if r.First {
		return deliverActions[0], true
	}
	i := r.rand.Intn(len(deliverActions))
	return deliverActions[i], true
}

func (r *OnlyDeliverPolicy) Update(_ *types.StepContext) {}
