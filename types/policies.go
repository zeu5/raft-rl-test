package types

import (
	"bytes"
	"encoding/json"
	"math"
	"math/rand"
	"os"
	"time"

	erand "golang.org/x/exp/rand"

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
	Update(*StepContext)
	// Reset at the end of running all the episodes
	// Used to invoke a cleanup of in memory resources
	Reset()
	// Record the QTable to a path in a specific format that can be parsed and assessed later
	Record(string)
}

// A fixed negative reward policy (-1) at all states
// The next action is chosen according to the softmax function
// With a temperature
type SoftMaxNegPolicy struct {
	QTable      map[string]map[string]float64
	Alpha       float64
	Gamma       float64
	Temperature float64

	rand erand.Source
}

// NewSoftMaxNegPolicy instantiated the SoftMaxNegPolicy
func NewSoftMaxNegPolicy(alpha, gamma, temperature float64) *SoftMaxNegPolicy {
	return &SoftMaxNegPolicy{
		QTable:      make(map[string]map[string]float64),
		Alpha:       alpha,
		Gamma:       gamma,
		Temperature: temperature,
		rand:        erand.NewSource(uint64(time.Now().UnixMilli())),
	}
}

// Checking interface compatibility
var _ Policy = &SoftMaxNegPolicy{}

func (s *SoftMaxNegPolicy) Record(path string) {
	bs := new(bytes.Buffer)

	for state, entries := range s.QTable {
		stateJ := make(map[string]interface{})
		stateJ["state"] = state
		stateJ["entries"] = entries

		stateBS, err := json.Marshal(stateJ)
		if err == nil {
			bs.Write(stateBS)
			bs.Write([]byte("\n"))
		}
	}

	if bs.Len() > 0 {
		os.WriteFile(path+".jsonl", bs.Bytes(), 0644)
	}
}

// Reset clears the QTable
func (s *SoftMaxNegPolicy) Reset() {
	s.QTable = make(map[string]map[string]float64)
	s.rand = erand.NewSource(uint64(time.Now().UnixMilli()))
}

func (s *SoftMaxNegPolicy) UpdateIteration(_ int, _ *Trace) {
	// if episode%200 == 0 {
	// 	fmt.Printf("Smallest value: %f\n", s.smallest)
	// }
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
	largestValue := s.QTable[stateHash][actions[0].Hash()]

	for i := 0; i < len(actions); i++ {
		action := actions[i]
		val := s.QTable[stateHash][action.Hash()]
		vals[i] = val
		if val > largestValue {
			largestValue = val
		}
	}

	// Normalizing
	for i := 0; i < len(vals); i++ {
		vals[i] = vals[i] - largestValue
		vals[i] = math.Exp(vals[i])
		sum += vals[i]
	}

	// Computing weights for each action
	for i, v := range vals {
		weights[i] = v / sum
	}
	// using the sampleuv library to sample based on the weights
	i, ok := sampleuv.NewWeighted(weights, s.rand).Take()
	if !ok {
		return nil, false
	}
	return actions[i], true
}

func (s *SoftMaxNegPolicy) Update(sCtx *StepContext) {
	state := sCtx.State
	action := sCtx.Action
	nextState := sCtx.NextState

	stateHash := state.Hash()

	nextStateHash := nextState.Hash()
	actionKey := action.Hash()
	if _, ok := s.QTable[stateHash]; !ok {
		s.QTable[stateHash] = make(map[string]float64)
	}
	if _, ok := s.QTable[stateHash][actionKey]; !ok {
		s.QTable[stateHash][actionKey] = 0
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
	nextVal := (1-s.Alpha)*curVal + s.Alpha*(-1+s.Gamma*max)
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

func (r *RandomPolicy) Record(_ string) {}

func (r *RandomPolicy) Reset() {}

func (r *RandomPolicy) UpdateIteration(_ int, _ *Trace) {}

func (r *RandomPolicy) NextAction(step int, state State, actions []Action) (Action, bool) {
	i := r.rand.Intn(len(actions))
	return actions[i], true
}

func (r *RandomPolicy) Update(_ *StepContext) {}
