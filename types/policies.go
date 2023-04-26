package types

import (
	"math"
	"math/rand"
	"time"

	"gonum.org/v1/gonum/stat/sampleuv"
)

type Policy interface {
	UpdateIteration(int, *Trace)
	NextAction(int, State, []Action) (Action, bool)
	Update(int, State, Action, State)
	Reset()
}

type SoftMaxNegPolicy struct {
	QTable map[string]map[string]float64
	alpha  float64
	gamma  float64
}

func NewSoftMaxNegPolicy(alpha, gamma float64) *SoftMaxNegPolicy {
	return &SoftMaxNegPolicy{
		QTable: make(map[string]map[string]float64),
		alpha:  alpha,
		gamma:  gamma,
	}
}

var _ Policy = &SoftMaxNegPolicy{}

func (s *SoftMaxNegPolicy) Reset() {
	s.QTable = make(map[string]map[string]float64)
}

func (s *SoftMaxNegPolicy) UpdateIteration(_ int, _ *Trace) {

}

func (s *SoftMaxNegPolicy) NextAction(step int, state State, actions []Action) (Action, bool) {
	stateHash := state.Hash()

	if _, ok := s.QTable[stateHash]; !ok {
		s.QTable[stateHash] = make(map[string]float64)
	}

	for _, a := range actions {
		aName := a.Hash()
		if _, ok := s.QTable[stateHash][aName]; !ok {
			s.QTable[stateHash][aName] = 0
		}
	}

	sum := float64(0)
	weights := make([]float64, len(actions))
	vals := make([]float64, len(actions))

	for i, action := range actions {
		val := s.QTable[stateHash][action.Hash()]
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

func (s *SoftMaxNegPolicy) Update(step int, state State, action Action, nextState State) {
	stateHash := state.Hash()

	nextStateHash := nextState.Hash()
	actionKey := action.Hash()
	if _, ok := s.QTable[stateHash]; !ok {
		return
	}
	if _, ok := s.QTable[stateHash][actionKey]; !ok {
		return
	}
	curVal := s.QTable[stateHash][actionKey]
	max := float64(0)
	if _, ok := s.QTable[nextStateHash]; ok {
		for _, val := range s.QTable[nextStateHash] {
			if val > max {
				max = val
			}
		}
	}
	nextVal := (1-s.alpha)*curVal + s.alpha*(-1+s.gamma*max)
	s.QTable[stateHash][actionKey] = nextVal
}

type RandomPolicy struct {
	rand *rand.Rand
}

var _ Policy = &RandomPolicy{}

func NewRandomPolicy() *RandomPolicy {
	return &RandomPolicy{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (r *RandomPolicy) Reset() {

}

func (r *RandomPolicy) UpdateIteration(_ int, _ *Trace) {

}

func (r *RandomPolicy) NextAction(step int, state State, actions []Action) (Action, bool) {
	i := r.rand.Intn(len(actions))
	return actions[i], true
}

func (r *RandomPolicy) Update(_ int, _ State, _ Action, _ State) {}
