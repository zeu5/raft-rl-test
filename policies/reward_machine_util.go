package policies

import "github.com/zeu5/raft-rl-test/types"

type RewardMachineDataset struct {
	segmentStates map[string]int
}

func RewardMachineAnalyzer(rm *RewardMachine) types.Analyzer {
	return func(s string, t []*types.Trace) types.DataSet {
		ds := &RewardMachineDataset{
			segmentStates: make(map[string]int),
		}
		for _, s := range rm.states {
			ds.segmentStates[s] = 0
		}
		// rmState := rm.states[0]
		// for _,

		return ds
	}
}

func RewardMachineCoverageComparator() types.Comparator {
	return func(s []string, ds []types.DataSet) {

	}
}
