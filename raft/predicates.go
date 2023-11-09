package raft

import (
	"github.com/zeu5/raft-rl-test/types"
	"go.etcd.io/raft/v3"
)

// File contains predicates over etcd-raft states

// return true if at least one of the replicas is in the leader state
func LeaderElectedPredicateState() types.RewardFuncSingle {
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

// return true if at least one of the replicas is in the leader state
func LeaderElectedPredicateNumber(elections int) types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if !ok {
			return false
		}

		for _, state := range pS.ReplicaStates { // for each replica state
			repState := state.(RaftReplicaState)
			curLog := repState.Log               // cast into raft.Status
			terms := make(map[uint64]bool, 0)    // set of unique terms
			filteredLog := filterEntries(curLog) // remove dummy entries
			for _, ent := range filteredLog {    // for each entry
				if len(ent.Data) == 0 {
					terms[ent.Term] = true // add its term to the set
				}
			}
			if len(terms) >= elections { // check number of unique terms
				return true
			}
		}

		return false
	}
}

// return true if all the replicas are not above the specified term number
func HighestTermForReplicas(term uint64) types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if !ok {
			return false
		}

		for _, state := range pS.ReplicaStates { // for each replica state
			repState := state.(RaftReplicaState)
			curState := repState.State // cast into raft.Status
			if curState.Term > term {
				return false
			}
		}
		return true
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
			if len(filterEntriesNoElection(curLog)) > 0 {
				return true
			}
		}

		return false
	}
}

// return true if there is at least one log with a single entry
func AtLeastOneLogOneEntry() types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if !ok {
			return false
		}

		for _, state := range pS.ReplicaStates { // for each replica state
			repState := state.(RaftReplicaState)
			curLog := repState.Log // cast into raft.Status
			if len(filterEntriesNoElection(curLog)) == 1 {
				return true
			}
		}

		return false
	}
}

// return true if there is at least one log with a single entry and a higher-term leader election entry
func AtLeastOneLogOneEntryPlusSubsequentLeaderElection() types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if !ok {
			return false
		}

		for _, state := range pS.ReplicaStates { // for each replica state
			repState := state.(RaftReplicaState)
			curLog := repState.Log // cast into raft.Status
			if len(filterEntriesNoElection(curLog)) == 1 {
				// take term of the committed entry
				committedEntryTerm := filterEntriesNoElection((curLog))[0].Term

				unfLog := filterEntries(curLog)
				for _, ent := range unfLog { // for each entry, included leader elections
					if ent.Term > committedEntryTerm {
						return true
					}
				}
			}
		}

		return false
	}
}

// return true if there is a log with at least one entry and a higher-term leader election entry
func AtLeastOneEntryANDSubsequentLeaderElection() types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if !ok {
			return false
		}

		for _, state := range pS.ReplicaStates { // for each replica state
			repState := state.(RaftReplicaState)
			curLog := repState.Log // cast into raft.Status
			if len(filterEntriesNoElection(curLog)) >= 1 {
				// take term of the committed entry
				committedEntryTerm := filterEntriesNoElection((curLog))[0].Term

				unfLog := filterEntries(curLog)
				for _, ent := range unfLog { // for each entry, included leader elections
					if ent.Term > committedEntryTerm {
						return true
					}
				}
			}
		}

		return false
	}
}

// return true if there is at least one log with a single entry and a replica with a term higher than the entry
func AtLeastOneLogOneEntryPlusReplicaInHigherTerm() types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if !ok {
			return false
		}

		for _, state := range pS.ReplicaStates { // for each replica state
			repState := state.(RaftReplicaState)
			curLog := repState.Log // cast into raft.Status
			if len(filterEntriesNoElection(curLog)) == 1 {
				// take term of the committed entry
				committedEntryTerm := filterEntriesNoElection((curLog))[0].Term

				for _, otherState := range pS.ReplicaStates { // check all replicas
					otherRepState := otherState.(RaftReplicaState)

					if otherRepState.State.Term > committedEntryTerm { // compare replica term with entry term
						return true
					}
				}
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

// return true if there are at least the specified number of requests in the stack
func StackSizeLowerBound(value int) types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if !ok {
			return false
		}

		return len(pS.PendingRequests) >= value
	}
}

func InState(state raft.StateType) types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, rs := range pS.ReplicaStates {
			repState := rs.(RaftReplicaState)
			if repState.State.RaftState == state {
				return true
			}
		}
		return false
	}
}

func InStateWithCommittedEntries(state raft.StateType, num int) types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, rs := range pS.ReplicaStates {
			repState := rs.(RaftReplicaState)
			curLog := repState.Log
			filteredLog := filterEntriesNoElection(curLog)
			if repState.State.RaftState == state && len(filteredLog) == num {
				return true
			}
		}
		return false
	}
}
