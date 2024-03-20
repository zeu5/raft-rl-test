package policies

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/zeu5/raft-rl-test/redisraft"
	"github.com/zeu5/raft-rl-test/types"
)

type RewardMachineDataset struct {
	// counters for the number of times each state is visited
	RMStateEpisodes  map[string]int // number of episodes each state is visited
	RMStateTimesteps map[string]int // number of timesteps spent in each state

	FinalPredicateStates map[string]bool // final state coverage
	TargetCoverage       []int           // number of unique states in the target space

	// abstracted
	AbsFinalPredicateStates map[string]bool // final state coverage
	AbsTargetCoverage       []int           // number of unique states in the target space

	// values of the first time the final state is reached
	FirstEpisodeToFinalPredicate  int
	FirstTimestepToFinalPredicate int

	// total counters
	totalEpisodes  int
	EpisodeHorizon int
}

// Takes in a sequence of predicates to analyze performance over
// Each trace is segmented based on the predicates to jump to
// The last segment contains the explored states

type RewardMachineAnalyzer struct {
	rmp *RewardMachinePolicy
	ds  *RewardMachineDataset

	reachedFinalPredicateOverall bool // true when the experiment has reached the final predicate at least once

	// for timesteps-based analysis
	EpisodeHorizon     int
	TracesToAnalyze    []*types.Trace
	TimestepsToAnalyze int

	// values for resuming halted episode analysis
	NextIndexOfFirstTrace  int
	LastRmStatePos         int
	FinalPredicateReached  bool
	EpisodeVisitedRMStates map[string]bool

	// abstraction
	Colors []redisraft.RedisRaftColorFunc
}

func RewardMachineAnalyzerCtor(rmp *RewardMachinePolicy, epHorizon int, colors ...redisraft.RedisRaftColorFunc) func() types.Analyzer {
	return func() types.Analyzer {
		return NewRewardMachineAnalyzer(rmp, epHorizon, colors...)
	}
}

func NewRewardMachineAnalyzer(rmp *RewardMachinePolicy, epHorizon int, colors ...redisraft.RedisRaftColorFunc) *RewardMachineAnalyzer {
	return &RewardMachineAnalyzer{
		rmp: rmp,
		ds: &RewardMachineDataset{
			RMStateEpisodes:  make(map[string]int),
			RMStateTimesteps: make(map[string]int),

			FinalPredicateStates: make(map[string]bool),
			TargetCoverage:       make([]int, 0),

			AbsFinalPredicateStates: make(map[string]bool),
			AbsTargetCoverage:       make([]int, 0),

			FirstEpisodeToFinalPredicate:  -1,
			FirstTimestepToFinalPredicate: -1,

			totalEpisodes:  0,
			EpisodeHorizon: epHorizon,
		},
		reachedFinalPredicateOverall: false,
		EpisodeHorizon:               epHorizon,
		TracesToAnalyze:              make([]*types.Trace, 0),
		TimestepsToAnalyze:           0,
		NextIndexOfFirstTrace:        0,
		LastRmStatePos:               0,
		FinalPredicateReached:        false,
		EpisodeVisitedRMStates:       make(map[string]bool),

		Colors: colors,
	}
}

func (rma *RewardMachineAnalyzer) Analyze(run int, episode int, startingTimestep int, s string, trace *types.Trace) {
	// fmt.Print("RM Analyze\n")
	// append to traces to analyze
	rma.TracesToAnalyze = append(rma.TracesToAnalyze, trace)
	rma.TimestepsToAnalyze += trace.Len()

	// fmt.Printf("RM to Analyze: %d\n", rma.TimestepsToAnalyze)

	// analyze a chunk of size horizon
	if rma.TimestepsToAnalyze >= rma.EpisodeHorizon {
		// fmt.Printf("RM Analyzing chunk\n")
		analyzed := 0                      // count of analyzed steps
		index := rma.NextIndexOfFirstTrace // index of the first unanalyzed step in the first trace of the buffer
		trace := rma.TracesToAnalyze[0]    // the first trace of the buffer

		curRmStatePos := rma.LastRmStatePos
		rmState := rma.rmp.rm.states[curRmStatePos]
		traceRMStatesVisited := rma.EpisodeVisitedRMStates
		finalPredicateReachedEpisode := rma.FinalPredicateReached

		for analyzed < rma.EpisodeHorizon { // until completing the chunk
			for analyzed < rma.EpisodeHorizon && index < trace.Len() { // until the end of trace or completed the chunk
				final := false                         // whether the current state is final or not
				_, _, nextState, _ := trace.Get(index) // at each timestep, take the nextState

				// update the spent timestep count for the current rm state (before updating the rm state)
				if _, ok := rma.ds.RMStateTimesteps[rmState]; !ok {
					rma.ds.RMStateTimesteps[rmState] = 0
				}
				// fmt.Printf("Adding a timestep to %s\n", rmState)
				rma.ds.RMStateTimesteps[rmState] += 1

				// update the rm state
				curRmStatePos, final = CheckRmCurrentState(rma.rmp.rm, nextState)

				// update the first time the final state is reached in the experiment
				if final && !rma.reachedFinalPredicateOverall {
					rma.ds.FirstTimestepToFinalPredicate = startingTimestep + index
					rma.reachedFinalPredicateOverall = true
				}

				// update if the final state is reached in the episode
				finalPredicateReachedEpisode = finalPredicateReachedEpisode || final

				// update the visited rm states in the episode (after updating the rm state with the nextState)
				rmState = rma.rmp.rm.states[curRmStatePos]
				if _, ok := traceRMStatesVisited[rmState]; !ok { // count unique rm states visited in the episode
					// fmt.Printf("RM State: %s visited\n", rmState)
					traceRMStatesVisited[rmState] = true
				}

				if final || (rma.rmp.oneTime && finalPredicateReachedEpisode) { // count unique final states explored, either still in predicate or oneTime machine
					nextStateHash := nextState.Hash()
					if _, ok := rma.ds.FinalPredicateStates[nextStateHash]; !ok { // count number of unique state explored in the target space (after reaching final state)
						rma.ds.FinalPredicateStates[nextStateHash] = true
					}

					// abstraction
					absState := redisraft.NewRedisPartState(nextState.(*types.Partition), rma.Colors...)
					absStateHash := absState.Hash()
					if _, ok := rma.ds.AbsFinalPredicateStates[absStateHash]; !ok { // count number of unique state explored in the target space (after reaching final state)
						rma.ds.AbsFinalPredicateStates[absStateHash] = true
					}

				}
				analyzed += 1 // update the analyzed steps count
				index += 1    // move to the next step
			}
			if index == trace.Len() { // episode ended, remove it from buffer and move to the next one
				// fmt.Printf("RM Analyzing episode ended\n")
				// fmt.Printf("RM States visited: %v\n", traceRMStatesVisited)
				// update counters for the number of episodes each rm state is visited
				for state := range traceRMStatesVisited {
					if _, ok := rma.ds.RMStateEpisodes[state]; !ok {
						rma.ds.RMStateEpisodes[state] = 0
					}
					// fmt.Printf("Adding an episode to %s\n", state)
					rma.ds.RMStateEpisodes[state] += 1
				}
				// eventually store the first episode reaching the final state
				if rma.ds.FirstEpisodeToFinalPredicate == -1 && rma.reachedFinalPredicateOverall {
					rma.ds.FirstEpisodeToFinalPredicate = episode
					// ds.TracesAfterFirstReached += 1
				}
				// update total episode count
				rma.ds.totalEpisodes += 1

				// move to next trace and reset analysis variables
				if len(rma.TracesToAnalyze) > 1 {
					rma.TracesToAnalyze = rma.TracesToAnalyze[1:]
					trace = rma.TracesToAnalyze[0]
					index = 0
					curRmStatePos = 0
					rmState = rma.rmp.rm.states[curRmStatePos]
					traceRMStatesVisited = make(map[string]bool)
					finalPredicateReachedEpisode = false
				}
			}
			if analyzed == rma.EpisodeHorizon { // analyzed all the steps of the chunk, stop here
				// fmt.Printf("RM chunk ended\n")

				// store the values to start from in the next chunk
				rma.NextIndexOfFirstTrace = index
				rma.LastRmStatePos = curRmStatePos
				rma.EpisodeVisitedRMStates = traceRMStatesVisited
				rma.FinalPredicateReached = finalPredicateReachedEpisode
			}
		}
		rma.TimestepsToAnalyze -= analyzed
		// fmt.Printf("RM Appending to coverage: %d\n", len(rma.ds.FinalPredicateStates))          // update the total number of timesteps in the buffer
		rma.ds.TargetCoverage = append(rma.ds.TargetCoverage, len(rma.ds.FinalPredicateStates)) // append the number of unique states in the overall list
		rma.ds.AbsTargetCoverage = append(rma.ds.AbsTargetCoverage, len(rma.ds.AbsFinalPredicateStates))
	}
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

		// target coverage
		// general
		// p := plot.New()

		// p.Title.Text = "Target Coverage: " + hierachyName
		// p.X.Label.Text = "Timesteps"
		// p.Y.Label.Text = "States covered"

		coverageData := make(map[string][]int)

		for i := 0; i < len(policies); i++ {
			name := policies[i]
			rmDS := ds[name].(*RewardMachineDataset)
			dataset := rmDS.TargetCoverage
			coverageData[policies[i]] = make([]int, len(dataset))
			copy(coverageData[policies[i]], dataset)
			// points := make(plotter.XYs, len(dataset))
			// for j, v := range dataset {
			// 	points[j] = plotter.XY{
			// 		X: float64(j) * float64(rmDS.EpisodeHorizon),
			// 		Y: float64(v),
			// 	}
			// }
			// line, err := plotter.NewLine(points)
			// if err != nil {
			// 	continue
			// }
			// line.Color = plotutil.Color(i)
			// p.Add(line)
			// p.Legend.Add(policies[i], line)
		}

		// p.Save(8*vg.Inch, 8*vg.Inch, path.Join(savePath, hierachyName+strconv.Itoa(run)+"_coverage.png"))

		bs, err = json.Marshal(coverageData)
		if err == nil {
			os.WriteFile(path.Join(savePath, hierachyName+strconv.Itoa(run)+"_data.json"), bs, 0644)
		}

		// abstraction
		// p = plot.New()

		// p.Title.Text = "Target Coverage (Colors): " + hierachyName
		// p.X.Label.Text = "Timesteps"
		// p.Y.Label.Text = "States covered"

		coverageData = make(map[string][]int)

		for i := 0; i < len(policies); i++ {
			name := policies[i]
			rmDS := ds[name].(*RewardMachineDataset)
			dataset := rmDS.AbsTargetCoverage
			coverageData[policies[i]] = make([]int, len(dataset))
			copy(coverageData[policies[i]], dataset)
			// points := make(plotter.XYs, len(dataset))
			// for j, v := range dataset {
			// 	points[j] = plotter.XY{
			// 		X: float64(j) * float64(rmDS.EpisodeHorizon),
			// 		Y: float64(v),
			// 	}
			// }
			// line, err := plotter.NewLine(points)
			// if err != nil {
			// 	continue
			// }
			// line.Color = plotutil.Color(i)
			// p.Add(line)
			// p.Legend.Add(policies[i], line)
		}

		// p.Save(8*vg.Inch, 8*vg.Inch, path.Join(savePath, hierachyName+strconv.Itoa(run)+"_abstractionCoverage.png"))

		bs, err = json.Marshal(coverageData)
		if err == nil {
			os.WriteFile(path.Join(savePath, hierachyName+strconv.Itoa(run)+"_abstraction_data.json"), bs, 0644)
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
	UpdateRm(int, types.State, types.Action, types.State, bool, bool, bool, int)
	// Update iteration with explicit reward flag, called after each transition
	UpdateIterationRm(int, *RMTrace, bool, int)
}

// To capture a trace segment
type RMTrace struct {
	steps      []int // step number in the episode
	states     []types.State
	actions    []types.Action
	nextStates []types.State
	rewards    []bool
	outOfSpace []bool

	AdditionalInfo map[string]interface{}
}

func NewRMTrace() *RMTrace {
	return &RMTrace{
		steps:      make([]int, 0),
		states:     make([]types.State, 0),
		actions:    make([]types.Action, 0),
		nextStates: make([]types.State, 0),
		rewards:    make([]bool, 0),
		outOfSpace: make([]bool, 0),

		AdditionalInfo: make(map[string]interface{}),
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
	t.steps = append(t.steps, step)
	t.states = append(t.states, state)
	t.actions = append(t.actions, action)
	t.nextStates = append(t.nextStates, nextState)
	t.rewards = append(t.rewards, reward)
	t.outOfSpace = append(t.outOfSpace, outOfSpace)
}

func (t *RMTrace) Len() int {
	return len(t.states)
}

func (t *RMTrace) Get(i int) (int, types.State, types.Action, types.State, bool, bool, bool) {
	if i >= len(t.states) {
		return -1, nil, nil, nil, false, false, false
	}
	return t.steps[i], t.states[i], t.actions[i], t.nextStates[i], t.rewards[i], t.outOfSpace[i], true
}

func (t *RMTrace) Last() (int, types.State, types.Action, types.State, bool, bool, bool) {
	if len(t.states) < 1 {
		return -1, nil, nil, nil, false, false, false
	}
	lastIndex := len(t.states) - 1
	return t.steps[lastIndex], t.states[lastIndex], t.actions[lastIndex], t.nextStates[lastIndex], t.rewards[lastIndex], t.outOfSpace[lastIndex], true
}

func (t *RMTrace) GetPrefix(i int) (*RMTrace, bool) {
	if i > len(t.states) {
		return nil, false
	}
	return &RMTrace{
		steps:      t.steps[0:i],
		states:     t.states[0:i],
		actions:    t.actions[0:i],
		nextStates: t.nextStates[0:i],
		rewards:    t.rewards[0:i],
		outOfSpace: t.outOfSpace[0:i],
	}, true
}
