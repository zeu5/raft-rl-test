package raft

import (
	"github.com/zeu5/raft-rl-test/types"
	"go.etcd.io/raft/v3"
	pb "go.etcd.io/raft/v3/raftpb"
)

// File contains predicates over etcd-raft states

// return true if at least one of the replicas is in the leader state
func LeaderElectedPredicate() types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if !ok {
			return false
		}

		for _, state := range pS.ReplicaStates { // for each replica state
			repState := state.(map[string]interface{})
			curState := repState["state"].(raft.Status) // cast into raft.Status
			if curState.BasicStatus.SoftState.RaftState.String() == "StateLeader" {
				return true
			}
		}

		return false
	}
}

// return true if the specified replica is in the leader state
func LeaderElectedPredicateSpecific(r_id uint64) types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if !ok {
			return false
		}

		state := pS.ReplicaStates[r_id] // take the replica state of the specified replica_id
		repState := state.(map[string]interface{})
		curState := repState["state"].(raft.Status) // cast into raft.Status
		return curState.BasicStatus.SoftState.RaftState.String() == "StateLeader"
	}
}

// return true if there is at least one entry in one of the replicas logs
func AtLeastOneLogNotEmpty() types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if !ok {
			return false
		}

		for _, state := range pS.ReplicaStates { // for each replica state
			repState := state.(map[string]interface{})
			curLog := repState["log"].([]pb.Entry) // cast into raft.Status
			if len(curLog) > 0 {
				return true
			}
		}

		return false
	}
}

// return true if the specified replica's log is empty
func EmptyLogSpecific(r_id uint64) types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if !ok {
			return false
		}

		state := pS.ReplicaStates[r_id] // take the replica state of the specified replica_id
		repState := state.(map[string]interface{})
		curLog := repState["log"].([]pb.Entry) // cast into raft.Status
		return len(curLog) == 0
	}
}
