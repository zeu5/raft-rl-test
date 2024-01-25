package cbft

import (
	"github.com/zeu5/raft-rl-test/types"
)

func AnyReachedRound(round int) types.RewardFuncSingle {
	return func(s types.State) bool {
		if s == nil {
			return false
		}
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if ns.Round >= round {
				return true
			}
		}
		return false
	}
}

func AnyReachedStep(step string) types.RewardFuncSingle {
	return func(s types.State) bool {
		if s == nil {
			return false
		}
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if ns.Step == step {
				return true
			}
		}
		return false
	}
}

func AllInRound(round int) types.RewardFuncSingle {
	return func(s types.State) bool {
		if s == nil {
			return false
		}
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if ns.Round != round {
				return false
			}
		}
		return true
	}
}

func AllAtLeastRound(round int) types.RewardFuncSingle {
	return func(s types.State) bool {
		if s == nil {
			return false
		}
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
		if s == nil {
			return false
		}
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
		if s == nil {
			return false
		}
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if ns.Height == height {
				return true
			}
		}
		return false
	}
}

func AnyAtLeastHeight(height int) types.RewardFuncSingle {
	return func(s types.State) bool {
		if s == nil {
			return false
		}
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if ns.Height >= height {
				return true
			}
		}
		return false
	}
}

func EmptyLockedForAll() types.RewardFuncSingle {
	return func(s types.State) bool {
		if s == nil {
			return false
		}
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if ns.LockedBlockHash != "" {
				return false
			}
		}
		return true
	}
}

func LockedForAll() types.RewardFuncSingle {
	return func(s types.State) bool {
		if s == nil {
			return false
		}
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if ns.LockedBlockHash == "" {
				return false
			}
		}
		return true
	}
}

func SameLockedForAll() types.RewardFuncSingle {
	return func(s types.State) bool {
		if s == nil {
			return false
		}
		lockedValues := make(map[string]bool)
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if ns.LockedBlockHash != "" {
				lockedValues[ns.LockedBlockHash] = true
			} else {
				return false
			}
		}
		return len(lockedValues) == 1
	}
}

func DifferentLocked() types.RewardFuncSingle {
	return func(s types.State) bool {
		if s == nil {
			return false
		}
		lockedValues := make(map[string]bool)
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if ns.LockedBlockHash != "" {
				lockedValues[ns.LockedBlockHash] = true
			}
		}
		return len(lockedValues) > 1
	}
}
