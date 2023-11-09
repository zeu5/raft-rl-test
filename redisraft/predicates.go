package redisraft

import (
	"strconv"

	"github.com/zeu5/raft-rl-test/types"
)

func LeaderElected() types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if !ok {
			return false
		}

		for _, state := range pS.ReplicaStates { // for each replica state
			repState := state.(*RedisNodeState)
			if repState.State == "leader" {
				return true
			}
		}

		return false
	}
}

func NumberCommit(commit int) types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, state := range ps.ReplicaStates {
			repState := state.(*RedisNodeState)
			if repState.Commit == commit {
				return true
			}
		}
		return false
	}
}

func TermNumber(term int) types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, state := range ps.ReplicaStates {
			repState := state.(*RedisNodeState)
			if repState.Term == term {
				return true
			}
		}
		return false
	}
}

func OnlyFollowersAndLeader() types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, state := range ps.ReplicaStates {
			repState := state.(*RedisNodeState)
			if repState.State != "leader" && repState.State != "follower" {
				return false
			}
		}
		return true
	}
}

func InState(state string) types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, state := range ps.ReplicaStates {
			repState := state.(*RedisNodeState)
			if repState.State == state {
				return true
			}
		}
		return false
	}
}

func CommitIndexAtLeast(idx int) types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, state := range ps.ReplicaStates {
			repState := state.(*RedisNodeState)
			if repState.Commit >= idx {
				return true
			}
		}
		return false
	}
}

func CurrentIndexAtLeast(idx int) types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, state := range ps.ReplicaStates {
			repState := state.(*RedisNodeState)
			if repState.Index >= idx {
				return true
			}
		}
		return false
	}
}

func NumConnectedNodesInAny(n int) types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, state := range ps.ReplicaStates {
			repState := state.(*RedisNodeState)
			connS, ok := repState.Params["num_nodes"]
			if !ok {
				continue
			}
			conn, err := strconv.Atoi(connS)
			if err != nil {
				continue
			}
			if conn > n {
				return true
			}
		}
		return false
	}
}
