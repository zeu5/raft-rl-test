package raft

import (
	"github.com/zeu5/raft-rl-test/types"
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
			repState := state.(RaftReplicaState)
			curState := repState.State // cast into raft.Status
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
		repState := state.(RaftReplicaState)
		curState := repState.State // cast into raft.Status
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
			repState := state.(RaftReplicaState)
			curLog := repState.Log // cast into raft.Status
			if len(filterEntries(curLog)) > 0 {
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
		repState := state.(RaftReplicaState)
		curLog := repState.Log // cast into raft.Status
		return len(filterEntries(curLog)) == 0
	}
}

// return true if there is at least one replica log with the specified number of entries (no more, dummy entries are ignored)
func ExactEntriesInLog(num int) types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if !ok {
			return false
		}

		for _, state := range pS.ReplicaStates { // for each replica state
			repState := state.(RaftReplicaState)
			curLog := repState.Log // cast into raft.Status
			if len(filterEntriesNoElection(curLog)) == num {
				return true
			}
		}

		return false
	}
}

// return true if the specified replica log has the specified number of entries (no more, dummy entries are ignored)
func ExactEntriesInLogSpecific(r_id uint64, num int) types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if !ok {
			return false
		}

		state := pS.ReplicaStates[r_id] // take the replica state of the specified replica_id
		repState := state.(RaftReplicaState)
		curLog := repState.Log // cast into raft.Status
		return len(filterEntriesNoElection(curLog)) == num
	}
}

// return true if there is at least one replica log with entries committed in, at least, the specified number of different terms (dummy entries are ignored)
func EntriesInDifferentTermsInLog(uniqueTerms int) types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if !ok {
			return false
		}

		for _, state := range pS.ReplicaStates { // for each replica state
			repState := state.(RaftReplicaState)
			curLog := repState.Log                         // cast into raft.Status
			terms := make(map[uint64]bool, 0)              // set of unique terms
			filteredLog := filterEntriesNoElection(curLog) // remove dummy entries
			for _, ent := range filteredLog {              // for each entry
				terms[ent.Term] = true // add its term to the set
			}
			if len(terms) >= uniqueTerms { // check number of unique terms
				return true
			}
		}

		return false
	}
}
