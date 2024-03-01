package redisraft

import "github.com/zeu5/raft-rl-test/types"

// returns true if at least one replica has a term greater than the given term
func MaxTerm(term int) func(*types.Partition) bool {
	return func(ps *types.Partition) bool {

		for _, state := range ps.ReplicaStates {
			repState := state.(*RedisNodeState)
			if repState.Term > term {
				return true
			}
		}
		return false
	}
}
