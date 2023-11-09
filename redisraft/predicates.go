package redisraft

import "github.com/zeu5/raft-rl-test/types"

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
