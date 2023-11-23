package cbft

import (
	"github.com/zeu5/raft-rl-test/types"
)

func AnyReachedRound(round int) types.RewardFuncSingle {
	return func(s types.State) bool {
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if ns.Round >= round {
				return true
			}
		}
		return false
	}
}

func AllAtLeastRound(round int) types.RewardFuncSingle {
	return func(s types.State) bool {
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if ns.Round < round {
				return false
			}
		}
		return true
	}
}

func AtLeastHeight(height int) types.RewardFuncSingle {
	return func(s types.State) bool {
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if ns.Height < height {
				return false
			}
		}
		return true
	}
}

func AnyAtHeight(height int) types.RewardFuncSingle {
	return func(s types.State) bool {
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if ns.Height == height {
				return true
			}
		}
		return false
	}
}
