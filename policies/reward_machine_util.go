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
	RMStateVisits                  map[string]int
	FinalPredicateStates           map[string]bool
	TracesAfterFirstReached        int
	FirstIterationToFinalPredicate int
}

// Takes in a sequence of predicates to analyze performance over
// Each trace is segmented based on the predicates to jump to
// The last segment contains the explored states

type RewardMachineAnalyzer struct {
	rmp *RewardMachinePolicy
	ds  *RewardMachineDataset
}

func NewRewardMachineAnalyzer(rmp *RewardMachinePolicy) *RewardMachineAnalyzer {
	return &RewardMachineAnalyzer{
		rmp: rmp,
		ds: &RewardMachineDataset{
			RMStateVisits:                  make(map[string]int),
			FinalPredicateStates:           make(map[string]bool),
			TracesAfterFirstReached:        0,
			FirstIterationToFinalPredicate: -1,
		},
	}
}

func (rma *RewardMachineAnalyzer) Analyze(run int, episode int, s string, trace *types.Trace) {
	rmp := rma.rmp
	rm := rmp.rm
	ds := rma.ds
	finalPredicateReached := false
	traceRMStatesVisited := make(map[string]bool)
	curRmStatePos := 0
	startState, _, _, _ := trace.Get(0)

	finalPredicate := rm.GetFinalPredicate()

	for j := len(rm.states) - 1; j >= 0; j-- {
		rmState := rm.states[j]
		predicate, ok := rm.predicates[rmState]
		if ok && predicate(startState) {
			curRmStatePos = j
			if rmState == FinalState {
				finalPredicateReached = true
			}
			break
		}
	}
	for i := 0; i < trace.Len(); i++ {
		_, _, nexState, _ := trace.Get(i)
		rmState := rm.states[curRmStatePos]
		if _, ok := traceRMStatesVisited[rmState]; !ok {
			traceRMStatesVisited[rmState] = true
		}
		for j := len(rm.states) - 1; j >= 0; j-- {
			rmState := rm.states[j]
			predicate, ok := rm.predicates[rmState]
			if ok && predicate(nexState) {
				if rmState == FinalState {
					finalPredicateReached = true
				}
				curRmStatePos = j
				break
			}
		}
		if finalPredicateReached && (rmp.oneTime || finalPredicate(nexState)) { // count unique final states explored, either still in predicate or oneTime machine
			nextStateHash := nexState.Hash()
			if _, ok := ds.FinalPredicateStates[nextStateHash]; !ok {
				ds.FinalPredicateStates[nextStateHash] = true
			}
		}
	}
	for state := range traceRMStatesVisited {
		if _, ok := ds.RMStateVisits[state]; !ok {
			ds.RMStateVisits[state] = 0
		}
		ds.RMStateVisits[state] += 1
	}

	if finalPredicateReached {
		if ds.FirstIterationToFinalPredicate == -1 {
			ds.FirstIterationToFinalPredicate = episode
		}
		ds.TracesAfterFirstReached += 1
	}
}

func (rma *RewardMachineAnalyzer) DataSet() types.DataSet {
	return rma.ds
}

func (rma *RewardMachineAnalyzer) Reset() {
	rma.ds = &RewardMachineDataset{
		RMStateVisits:                  make(map[string]int),
		FinalPredicateStates:           make(map[string]bool),
		TracesAfterFirstReached:        0,
		FirstIterationToFinalPredicate: -1,
	}
}

var _ types.Analyzer = (*RewardMachineAnalyzer)(nil)

// returns a comparator for a predHierarchy analyzer.
func RewardMachineCoverageComparator(savePath string, hierachyName string) types.Comparator {
	if _, err := os.Stat(savePath); err != nil {
		os.Mkdir(savePath, os.ModePerm)
	}
	return func(run, episodes int, policies []string, ds []types.DataSet) {
		// readable text file
		printable := ""
		for i := 0; i < len(ds); i++ {
			printable = printable + fmt.Sprintf("For run: %d, experiment: %s\n", run, policies[i])
			rmDS := ds[i].(*RewardMachineDataset)
			for state, count := range rmDS.RMStateVisits {
				printable = printable + fmt.Sprintf("\tRM State %s, Visits: %d\n", state, count)
			}
			printable = printable + fmt.Sprintf("\tFinal predicate states: %d\n", len(rmDS.FinalPredicateStates))
			printable = printable + fmt.Sprintf("\tFirst iteration to Final predicate: %d\n", rmDS.FirstIterationToFinalPredicate)
			repeatAccuracy := float64(rmDS.TracesAfterFirstReached) / float64(episodes-rmDS.FirstIterationToFinalPredicate)
			printable = printable + fmt.Sprintf("\tTraces repeat accuracy after reaching final states: %f\n", repeatAccuracy)
		}
		os.WriteFile(path.Join(savePath, hierachyName+"_"+strconv.Itoa(run)+".txt"), []byte(printable), 0644)

		// json file
		data := make(map[string]interface{})
		for i, policyName := range policies {
			rmDS := ds[i].(*RewardMachineDataset)
			d := make(map[string]interface{})
			d["rmStateVisits"] = rmDS.RMStateVisits
			d["finalPredicateStates"] = len(rmDS.FinalPredicateStates)
			d["repeatAccuracy"] = float64(rmDS.TracesAfterFirstReached) / float64(episodes-rmDS.FirstIterationToFinalPredicate)
			d["firstIterationToFinalPredicate"] = rmDS.FirstIterationToFinalPredicate
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

func (pa *PredicatesAnalyzer) Analyze(run int, episode int, s string, trace *types.Trace) {
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
	return func(run, _ int, s []string, ds []types.DataSet) {
		for i := 0; i < len(s); i++ {
			fmt.Printf("For run: %d, experiment: %s\n", run, s[i])
			pDS := ds[i].(*predicatesDataset)
			for s, count := range pDS.predicates {
				fmt.Printf("\tPredicate: %d, num of times satisfied: %d\n", s, count)
			}
		}
	}
}
