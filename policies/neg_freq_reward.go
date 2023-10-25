package policies

import "github.com/zeu5/raft-rl-test/types"

type SoftMaxNegFreqPolicy struct {
	*types.SoftMaxNegPolicy
	Freq map[string]int
}

func NewSoftMaxNegFreqPolicy(alpha, gamma, temp float64) *SoftMaxNegFreqPolicy {
	return &SoftMaxNegFreqPolicy{
		SoftMaxNegPolicy: types.NewSoftMaxNegPolicy(alpha, gamma, temp),
		Freq:             make(map[string]int),
	}
}

func (t *SoftMaxNegFreqPolicy) Update(step int, state types.State, action types.Action, nextState types.State) {
	stateHash := state.Hash()

	nextStateHash := nextState.Hash()
	actionKey := action.Hash()
	if _, ok := t.QTable[stateHash]; !ok {
		t.QTable[stateHash] = make(map[string]float64)
	}
	if _, ok := t.QTable[stateHash][actionKey]; !ok {
		t.QTable[stateHash][actionKey] = 0
	}
	curVal := t.QTable[stateHash][actionKey]
	max := float64(0)
	if _, ok := t.QTable[nextStateHash]; ok {
		for _, val := range t.QTable[nextStateHash] {
			if val > max {
				max = val
			}
		}
	}
	if _, ok := t.Freq[nextStateHash]; !ok {
		t.Freq[nextStateHash] = 0
	}
	t.Freq[nextStateHash] += 1
	reward := float64(-1 * t.Freq[nextStateHash])

	// the update with -1 reward
	nextVal := (1-t.Alpha)*curVal + t.Alpha*(reward+t.Gamma*max)
	t.QTable[stateHash][actionKey] = nextVal
}

func (t *SoftMaxNegFreqPolicy) Reward(sHash string) float64 {
	return float64(t.Freq[sHash] * -1)
}
