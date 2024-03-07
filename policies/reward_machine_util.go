package policies

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/zeu5/raft-rl-test/types"
)

type RewardMachineDataset struct {
	// counters for the number of times each state is visited
	RMStateEpisodes  map[string]int // number of episodes each state is visited
	RMStateTimesteps map[string]int // number of timesteps spent in each state

	FinalPredicateStates map[string]bool // final state coverage

	// values of the first time the final state is reached
	FirstEpisodeToFinalPredicate  int
	FirstTimestepToFinalPredicate int

	// total counters
	totalEpisodes int
}

// Takes in a sequence of predicates to analyze performance over
// Each trace is segmented based on the predicates to jump to
// The last segment contains the explored states

type RewardMachineAnalyzer struct {
	rmp *RewardMachinePolicy
	ds  *RewardMachineDataset
}

func RewardMachineAnalyzerCtor(rmp *RewardMachinePolicy) func() types.Analyzer {
	return func() types.Analyzer {
		return NewRewardMachineAnalyzer(rmp)
	}
}

func NewRewardMachineAnalyzer(rmp *RewardMachinePolicy) *RewardMachineAnalyzer {
	return &RewardMachineAnalyzer{
		rmp: rmp,
		ds: &RewardMachineDataset{
			RMStateEpisodes:  make(map[string]int),
			RMStateTimesteps: make(map[string]int),

			FinalPredicateStates: make(map[string]bool),

			FirstEpisodeToFinalPredicate:  -1,
			FirstTimestepToFinalPredicate: -1,

			totalEpisodes: 0,
		},
	}
}

func (rma *RewardMachineAnalyzer) Analyze(run int, episode int, startingTimestep int, s string, trace *types.Trace) {
	rmp := rma.rmp // reward machine policy
	rm := rmp.rm   // reward machine
	ds := rma.ds   // dataset
	finalPredicateReached := false
	traceRMStatesVisited := make(map[string]bool)
	curRmStatePos := 0

	finalPredicate := rm.GetFinalPredicate()
	final := false

	// first step of the trace
	startState, _, _, _ := trace.Get(0)
	curRmStatePos, final = CheckRmCurrentState(rm, startState)
	rmState := rm.states[curRmStatePos]
	if _, ok := traceRMStatesVisited[rmState]; !ok { // count unique rm states visited in the episode
		traceRMStatesVisited[rmState] = true
	}
	if final && !finalPredicateReached {
		ds.FirstTimestepToFinalPredicate = startingTimestep
		finalPredicateReached = true
	}

	for i := 0; i < trace.Len(); i++ { // iterate through the trace, all states except the first one
		_, _, nextState, _ := trace.Get(i) // at each timestep, take the nextState

		// update the spent timestep count for the current rm state (before updating the rm state)
		if _, ok := ds.RMStateTimesteps[rmState]; !ok {
			ds.RMStateTimesteps[rmState] = 0
		}
		ds.RMStateTimesteps[rmState] += 1

		// update the rm state
		curRmStatePos, final = CheckRmCurrentState(rm, nextState)
		if final && !finalPredicateReached {
			ds.FirstTimestepToFinalPredicate = startingTimestep + i
			finalPredicateReached = true
		}

		// update the visited rm states in the episode (after updating the rm state with the nextState)
		rmState = rm.states[curRmStatePos]
		if _, ok := traceRMStatesVisited[rmState]; !ok { // count unique rm states visited in the episode
			traceRMStatesVisited[rmState] = true
		}

		if finalPredicateReached && (rmp.oneTime || finalPredicate(nextState)) { // count unique final states explored, either still in predicate or oneTime machine
			nextStateHash := nextState.Hash()
			if _, ok := ds.FinalPredicateStates[nextStateHash]; !ok { // count number of unique state explored in the target space (after reaching final state)
				ds.FinalPredicateStates[nextStateHash] = true
			}
		}
	}
	// update counters for the number of episodes each rm state is visited
	for state := range traceRMStatesVisited {
		if _, ok := ds.RMStateEpisodes[state]; !ok {
			ds.RMStateEpisodes[state] = 0
		}
		ds.RMStateEpisodes[state] += 1
	}

	// eventually store the first episode reaching the final state
	if ds.FirstEpisodeToFinalPredicate == -1 && finalPredicateReached {
		ds.FirstEpisodeToFinalPredicate = episode
		// ds.TracesAfterFirstReached += 1
	}
	// update total episode count
	ds.totalEpisodes += 1
}

// takes a reward machine and a state and returns the position of the current state of the reward machine and whether that is the final state or not
func CheckRmCurrentState(rm *RewardMachine, state types.State) (int, bool) {
	finalPredicateReached := false
	curRmStatePos := 0

	for j := len(rm.states) - 1; j >= 0; j-- {
		rmState := rm.states[j]
		predicate, ok := rm.predicates[rmState]
		if ok && predicate(state) {
			if rmState == FinalState {
				finalPredicateReached = true
			}
			curRmStatePos = j
			break
		}
	}
	return curRmStatePos, finalPredicateReached
}

func (rma *RewardMachineAnalyzer) DataSet() types.DataSet {
	return rma.ds
}

func (rma *RewardMachineAnalyzer) Reset() {
	rma.ds = &RewardMachineDataset{
		RMStateEpisodes:  make(map[string]int),
		RMStateTimesteps: make(map[string]int),

		FinalPredicateStates: make(map[string]bool),

		FirstEpisodeToFinalPredicate:  -1,
		FirstTimestepToFinalPredicate: -1,

		totalEpisodes: 0,
	}
}

var _ types.Analyzer = (*RewardMachineAnalyzer)(nil)

// returns a comparator for a predHierarchy analyzer.
func RewardMachineCoverageComparator(savePath string, hierachyName string) types.Comparator {
	if _, err := os.Stat(savePath); err != nil {
		os.Mkdir(savePath, os.ModePerm)
	}
	return func(run, _ int, policies []string, ds map[string]types.DataSet) {
		// readable text file
		printable := ""
		for i := 0; i < len(policies); i++ {
			name := policies[i]
			printable = printable + fmt.Sprintf("For run: %d, experiment: %s\n", run, policies[i])
			rmDS := ds[name].(*RewardMachineDataset)
			episodes := rmDS.totalEpisodes
			for state, count := range rmDS.RMStateEpisodes {
				printable = printable + fmt.Sprintf("\tRM State %s, Visits: %d, Timesteps spent: %d\n", state, count, rmDS.RMStateTimesteps[state])
			}
			printable = printable + fmt.Sprintf("\tFinal predicate unique states: %d\n", len(rmDS.FinalPredicateStates))

			printable = printable + fmt.Sprintf("\tFirst episode to Final predicate: %d\n", rmDS.FirstEpisodeToFinalPredicate)
			printable = printable + fmt.Sprintf("\tFirst timestep to Final predicate: %d\n", rmDS.FirstTimestepToFinalPredicate)

			repeatAccuracy := float64(rmDS.RMStateEpisodes["Final"]) / float64(episodes-rmDS.FirstEpisodeToFinalPredicate)
			printable = printable + fmt.Sprintf("\tTraces repeat accuracy after reaching final states: %f\n", repeatAccuracy)
		}
		os.WriteFile(path.Join(savePath, hierachyName+"_"+strconv.Itoa(run)+".txt"), []byte(printable), 0644)

		// json file
		data := make(map[string]interface{})
		for _, policyName := range policies {
			rmDS := ds[policyName].(*RewardMachineDataset)
			episodes := rmDS.totalEpisodes
			d := make(map[string]interface{})
			d["rmStateVisits"] = rmDS.RMStateEpisodes
			d["finalPredicateStates"] = len(rmDS.FinalPredicateStates)
			d["repeatAccuracy"] = float64(rmDS.RMStateEpisodes["Final"]) / float64(episodes-rmDS.FirstEpisodeToFinalPredicate)
			d["firstEpisodeToFinalPredicate"] = rmDS.FirstEpisodeToFinalPredicate
			data[policyName] = d
		}
		bs, err := json.Marshal(data)
		if err == nil {
			os.WriteFile(path.Join(savePath, hierachyName+"_"+strconv.Itoa(run)+".json"), bs, 0644)
		}
	}
}

type predicatesDataset struct {
	predicates map[int]int
}

type PredicatesAnalyzer struct {
	predicates []types.RewardFuncSingle
	satisfied  map[int]int
}

func NewPredicatesAnalyzer(predicates ...types.RewardFuncSingle) *PredicatesAnalyzer {
	return &PredicatesAnalyzer{
		predicates: predicates,
		satisfied:  make(map[int]int),
	}
}

func (pa *PredicatesAnalyzer) Analyze(run int, episode int, startingTimestep int, s string, trace *types.Trace) {
	for i := 0; i < trace.Len(); i++ {
		state, _, _, _ := trace.Get(i)
		for j, p := range pa.predicates {
			if p(state) {
				count, ok := pa.satisfied[j]
				if !ok {
					pa.satisfied[j] = 0
					count = 0
				}
				pa.satisfied[j] = count + 1
			}
		}
	}
}

func (pa *PredicatesAnalyzer) DataSet() types.DataSet {
	out := make(map[int]int)
	for s, count := range pa.satisfied {
		out[s] = count
	}
	return out
}

func (pa *PredicatesAnalyzer) Reset() {
	pa.satisfied = make(map[int]int)
}

var _ types.Analyzer = (*PredicatesAnalyzer)(nil)

func PredicatesComparator() types.Comparator {
	return func(run, _ int, s []string, ds map[string]types.DataSet) {
		for i := 0; i < len(s); i++ {
			name := s[i]
			fmt.Printf("For run: %d, experiment: %s\n", run, s[i])
			pDS := ds[name].(*predicatesDataset)
			for s, count := range pDS.predicates {
				fmt.Printf("\tPredicate: %d, num of times satisfied: %d\n", s, count)
			}
		}
	}
}

// Generic RM Policy interface
type RMPolicy interface {
	types.Policy
	// Update with explicit reward flag, called after each transition
	UpdateRm(int, types.State, types.Action, types.State, bool, bool)
	// Update iteration with explicit reward flag, called after each transition
	UpdateIterationRm(int, *RMTrace)
}

// To capture a trace segment
type RMTrace struct {
	states     []types.State
	actions    []types.Action
	nextStates []types.State
	rewards    []bool
	outOfSpace []bool
}

func NewRMTrace() *RMTrace {
	return &RMTrace{
		states:     make([]types.State, 0),
		actions:    make([]types.Action, 0),
		nextStates: make([]types.State, 0),
		rewards:    make([]bool, 0),
		outOfSpace: make([]bool, 0),
	}
}

func (t *RMTrace) Slice(from, to int) *RMTrace {
	slicedTrace := NewRMTrace()
	for i := from; i < to; i++ {
		slicedTrace.Append(i-from, t.states[i], t.actions[i], t.nextStates[i], t.rewards[i], t.outOfSpace[i])
	}
	return slicedTrace
}

func (t *RMTrace) Append(step int, state types.State, action types.Action, nextState types.State, reward bool, outOfSpace bool) {
	t.states = append(t.states, state)
	t.actions = append(t.actions, action)
	t.nextStates = append(t.nextStates, nextState)
	t.rewards = append(t.rewards, reward)
	t.outOfSpace = append(t.outOfSpace, outOfSpace)
}

func (t *RMTrace) Len() int {
	return len(t.states)
}

func (t *RMTrace) Get(i int) (types.State, types.Action, types.State, bool, bool, bool) {
	if i >= len(t.states) {
		return nil, nil, nil, false, false, false
	}
	return t.states[i], t.actions[i], t.nextStates[i], t.rewards[i], t.outOfSpace[i], true
}

func (t *RMTrace) Last() (types.State, types.Action, types.State, bool, bool, bool) {
	if len(t.states) < 1 {
		return nil, nil, nil, false, false, false
	}
	lastIndex := len(t.states) - 1
	return t.states[lastIndex], t.actions[lastIndex], t.nextStates[lastIndex], t.rewards[lastIndex], t.outOfSpace[lastIndex], true
}

func (t *RMTrace) GetPrefix(i int) (*RMTrace, bool) {
	if i > len(t.states) {
		return nil, false
	}
	return &RMTrace{
		states:     t.states[0:i],
		actions:    t.actions[0:i],
		nextStates: t.nextStates[0:i],
		rewards:    t.rewards[0:i],
		outOfSpace: t.outOfSpace[0:i],
	}, true
}
