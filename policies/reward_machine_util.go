package policies

import (
	"fmt"

	"github.com/zeu5/raft-rl-test/types"
)

type RewardMachineDataset struct {
	rmStateVisits map[string]int
}

// Takes in a sequence of predicates to analyze performance over
// Each trace is segmented based on the predicates to jump to
// The last segment contains the explored states

func RewardMachineAnalyzer(rm *RewardMachine) types.Analyzer {
	return func(run int, s string, traces []*types.Trace) types.DataSet {
		ds := &RewardMachineDataset{
			rmStateVisits: make(map[string]int),
		}

		for _, t := range traces {
			traceRMStatesVisited := make(map[string]bool)
			curRmStatePos := 0
			startState, _, _, _ := t.Get(0)
			for j := len(rm.states) - 1; j >= 0; j-- {
				rmState := rm.states[j]
				predicate, ok := rm.predicates[rmState]
				if ok && predicate(startState) {
					curRmStatePos = j
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
						curRmStatePos = j
						break
					}
				}
			}
			for state := range traceRMStatesVisited {
				if _, ok := ds.rmStateVisits[state]; !ok {
					ds.rmStateVisits[state] = 0
				}
				ds.rmStateVisits[state] += 1
			}
		}

		return ds
	}
}

func RewardMachineCoverageComparator() types.Comparator {
	return func(run int, s []string, ds []types.DataSet) {
		for i := 0; i < len(ds); i++ {
			fmt.Printf("For run: %d, experiment: %s\n", run, s[i])
			rmDS := ds[i].(*RewardMachineDataset)
			for state, count := range rmDS.rmStateVisits {
				fmt.Printf("\tRM State %s, Visits: %d\n", state, count)
			}
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
