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
	RepeatAccuracy                 float64
	FirstIterationToFinalPredicate int
}

// Takes in a sequence of predicates to analyze performance over
// Each trace is segmented based on the predicates to jump to
// The last segment contains the explored states

func RewardMachineAnalyzer(rm *RewardMachine) types.Analyzer {
	return func(run int, s string, traces []*types.Trace) types.DataSet {
		ds := &RewardMachineDataset{
			RMStateVisits:                  make(map[string]int),
			FinalPredicateStates:           make(map[string]bool),
			RepeatAccuracy:                 0,
			FirstIterationToFinalPredicate: -1,
		}

		finalPredicate := rm.GetFinalPredicate()
		numTracesFinalReached := 0

		for traceNo, t := range traces {
			finalPredicateReached := false

			traceRMStatesVisited := make(map[string]bool)
			curRmStatePos := 0
			startState, _, _, _ := t.Get(0)
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
			for i := 0; i < t.Len(); i++ {
				_, _, nexState, _ := t.Get(i)
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
				if finalPredicateReached && finalPredicate(nexState) {
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
					ds.FirstIterationToFinalPredicate = traceNo
				}
				numTracesFinalReached += 1
			}
		}

		if ds.FirstIterationToFinalPredicate != -1 {
			afterFirstReached := len(traces) - ds.FirstIterationToFinalPredicate
			ds.RepeatAccuracy = float64(numTracesFinalReached) / float64(afterFirstReached)
		}

		return ds
	}
}

func RewardMachineCoverageComparator(savePath string) types.Comparator {
	if _, err := os.Stat(savePath); err != nil {
		os.Mkdir(savePath, os.ModePerm)
	}
	return func(run int, s []string, ds []types.DataSet) {
		for i := 0; i < len(ds); i++ {
			fmt.Printf("For run: %d, experiment: %s\n", run, s[i])
			rmDS := ds[i].(*RewardMachineDataset)
			for state, count := range rmDS.RMStateVisits {
				fmt.Printf("\tRM State %s, Visits: %d\n", state, count)
			}
			fmt.Printf("\tFinal predicate states: %d\n", len(rmDS.FinalPredicateStates))
			fmt.Printf("\tFirst iteration to Final predicate: %d\n", rmDS.FirstIterationToFinalPredicate)
			fmt.Printf("\tTraces repeat accuracy after reaching final states: %f\n", rmDS.RepeatAccuracy)
		}
		data := make(map[string]interface{})
		for i, b := range s {
			rmDS := ds[i].(*RewardMachineDataset)
			d := make(map[string]interface{})
			d["rmStateVisits"] = rmDS.RMStateVisits
			d["finalPredicateStates"] = len(rmDS.FinalPredicateStates)
			d["repeatAccuracy"] = rmDS.RepeatAccuracy
			d["firstiterationToFinalPredicate"] = rmDS.FirstIterationToFinalPredicate
			data[b] = d
		}
		bs, err := json.Marshal(data)
		if err == nil {
			os.WriteFile(path.Join(savePath, strconv.Itoa(run)+".json"), bs, 0644)
		}
	}
}

type predicatesDataset struct {
	predicates map[int]int
}

func PredicatesAnalyzer(predicates ...types.RewardFuncSingle) types.Analyzer {
	return func(run int, s string, traces []*types.Trace) types.DataSet {
		d := &predicatesDataset{
			predicates: make(map[int]int),
		}
		for _, t := range traces {
			for i := 0; i < t.Len(); i++ {
				s, _, _, _ := t.Get(i)
				for j, p := range predicates {
					if p(s) {
						count, ok := d.predicates[j]
						if !ok {
							d.predicates[j] = 0
							count = 0
						}
						d.predicates[j] = count + 1
					}
				}
			}
		}
		return d
	}
}

func PredicatesComparator() types.Comparator {
	return func(run int, s []string, ds []types.DataSet) {
		for i := 0; i < len(s); i++ {
			fmt.Printf("For run: %d, experiment: %s\n", run, s[i])
			pDS := ds[i].(*predicatesDataset)
			for s, count := range pDS.predicates {
				fmt.Printf("\tPredicate: %d, num of times satisfied: %d\n", s, count)
			}
		}
	}
}
