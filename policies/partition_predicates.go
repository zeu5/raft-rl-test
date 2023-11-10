package policies

import "github.com/zeu5/raft-rl-test/types"

func NoPartition() types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		return len(ps.Partition) == 1
	}
}
