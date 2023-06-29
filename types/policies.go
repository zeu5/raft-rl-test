package types

import (
	"math"
	"math/rand"
	"time"

	"gonum.org/v1/gonum/stat/sampleuv"
)

// Generic Policy interface
type Policy interface {
	// Update at the end of the episode with the complete trace
	UpdateIteration(int, *Trace)
	// Policy should return the next action to take at a given state
	// The second return param indicates if the policy did choose an action
	NextAction(int, State, []Action) (Action, bool)
	// Update called after each transition
	Update(int, State, Action, State)
	// Reset at the end of running all the episodes
	// Used to invoke a cleanup of in memory resources
	Reset()
}

// Generic RM Policy interface
type RmPolicy interface {
	Policy
	// Update with explicit reward flag, called after each transition
	UpdateRm(int, State, Action, State, bool, bool)
}

// A fixed negative reward policy (-1) at all states
// The next action is chosen according to the softmax function
// With a temperature
type SoftMaxNegPolicy struct {
	QTable      map[string]map[string]float64
	alpha       float64
	gamma       float64
	temperature float64
}

// NewSoftMaxNegPolicy instantiated the SoftMaxNegPolicy
func NewSoftMaxNegPolicy(alpha, gamma, temperature float64) *SoftMaxNegPolicy {
	return &SoftMaxNegPolicy{
		QTable:      make(map[string]map[string]float64),
		alpha:       alpha,
		gamma:       gamma,
		temperature: temperature,
	}
}

// Checking interface compatibility
var _ Policy = &SoftMaxNegPolicy{}

// Reset clears the QTable
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

	// Initializing QTable entry to 0 if it does not exist
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
		val := s.QTable[stateHash][action.Hash()] * (1 / s.temperature)
		exp := math.Exp(val)
		vals[i] = exp
		sum += exp
	}

	// Computing weights for each action
	for i, v := range vals {
		weights[i] = v / sum
	}
	// using the sampleuv library to sample based on the weights
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
	// the update with -1 reward
	nextVal := (1-s.alpha)*curVal + s.alpha*(-1+s.gamma*max)
	s.QTable[stateHash][actionKey] = nextVal
}

// RandomPolicy to choose the next action purely randomly
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
