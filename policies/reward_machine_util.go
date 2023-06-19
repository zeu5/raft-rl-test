package policies

import (
	"fmt"

	"github.com/zeu5/raft-rl-test/types"
)

type RewardMachineDataset struct {
	segmentStates  map[int]int
	predicateStats map[int]int
}

// Takes in a sequence of predicates to analyze performance over
// Each trace is segmented based on the predicates to jump to
// The last segment contains the explored states

func RewardMachineAnalyzer(predicates []types.RewardFunc, abs types.StateAbstractor) types.Analyzer {
	return func(s string, traces []*types.Trace) types.DataSet {
		segmentStatesMap := make(map[int]map[string]bool)
		ds := &RewardMachineDataset{
			segmentStates:  make(map[int]int),
			predicateStats: make(map[int]int),
		}
		for i := -1; i < len(predicates); i++ {
			segmentStatesMap[i] = make(map[string]bool)
		}

		for _, t := range traces {
			curPred := -1
			for i := 0; i < t.Len(); i++ {
				state, _, nextState, _ := t.Get(i)
				stateHash := abs(state)
				if _, ok := segmentStatesMap[curPred][stateHash]; !ok {
					segmentStatesMap[curPred][stateHash] = true
				}

				for j := len(predicates) - 1; j > curPred; j-- {
					pred := predicates[j]
					if pred(state, nextState) {
						curPred = j
						break
					}
				}
			}
		}

		for p, states := range segmentStatesMap {
			ds.segmentStates[p] = len(states)
		}

		return ds
	}
}

func RewardMachineCoverageComparator() types.Comparator {
	return func(s []string, ds []types.DataSet) {
		for i := 0; i < len(ds); i++ {
			fmt.Printf("For experiment: %s\n", s[i])
			rmDS := ds[i].(*RewardMachineDataset)
			for p, states := range rmDS.segmentStates {
				fmt.Printf("\tPredicate %d, States: %d\n", p+1, states)
			}
		}
	}
}
