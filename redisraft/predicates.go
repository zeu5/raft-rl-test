package redisraft

import (
	"math"
	"strconv"

	"github.com/zeu5/raft-rl-test/types"
)

// returns true if one replica is in the state leader
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

// returns true if at least one replica has the specified commit value
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

// returns true if at least one replica has the specified term value
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

// returns true if all the replicas are either in leader or follower state
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

// returns true if at least one replica is in the specified state
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

// returns true if at least one replica has commit value greater or equal to the specified one
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

// returns true if at least one replica has index value greater or equal to the specified one
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

// returns true if at least one replica has the specified number of connected nodes (?)
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
			if conn >= n {
				return true
			}
		}
		return false
	}
}

// returns true if there are two replicas with terms difference greater or equal to the specified one
func OutOfSyncBy(diff int) types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		maxTerm := math.MinInt
		minTerm := math.MaxInt

		for _, state := range ps.ReplicaStates {
			repState := state.(*RedisNodeState)
			if repState.Term > maxTerm {
				maxTerm = repState.Term
			}
			if repState.Term < minTerm {
				minTerm = repState.Term
			}
		}
		return (maxTerm - minTerm) >= diff
	}
}

// returns true if all the replicas are in the same term, and this term is greater or equal than the specified one
func AllInSyncAtleast(minTerm int) types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		allTerms := make(map[int]bool)
		for _, state := range ps.ReplicaStates {
			rState := state.(*RedisNodeState)
			allTerms[rState.Term] = true
			if rState.Term < minTerm {
				return false
			}
		}
		return len(allTerms) == 1
	}
}

// returns true if there is no replica with term below the specified value
func AllInTermAtleast(minTerm int) types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, state := range ps.ReplicaStates {
			rState := state.(*RedisNodeState)
			if rState.Term < minTerm {
				return false
			}
		}
		return true
	}
}

// returns true if there is at least one log with entries in two different terms, consider only NORMAL entries
func EntriesInDifferentTerms() types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, state := range ps.ReplicaStates {
			rState := state.(*RedisNodeState)
			terms := make(map[int]bool)
			for _, entry := range rState.Logs {
				if EntryTypeToString(entry.Type) == "NORMAL" {
					terms[entry.Term] = true
				}
			}
			if len(terms) > 1 {
				return true
			}
		}
		return false
	}
}

// returns true if all nodes have at least the given commit value.
//
// PROBLEM if we work with crashes?
func NodesCommitValuesAtLeast(val int) types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, state := range ps.ReplicaStates {
			rState := state.(*RedisNodeState)
			if rState.Commit < val {
				return false
			}
		}
		return true
	}
}

// returns true if all nodes have at most the given commit value.
func NodesCommitValuesAtMost(val int) types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, state := range ps.ReplicaStates {
			rState := state.(*RedisNodeState)
			if rState.Commit > val {
				return false
			}
		}
		return true
	}
}

// returns true if there is at least one NORMAL entry in any of the logs. Also not committed.
func AtLeastOneNormalEntry() types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, state := range ps.ReplicaStates {
			rState := state.(*RedisNodeState)
			for _, entry := range rState.Logs {
				if EntryTypeToString(entry.Type) == "NORMAL" {
					return true
				}
			}
		}
		return false
	}
}

// returns true if there is at least one committed NORMAL entry in any of the logs.
func AtLeastOneCommittedNormalEntry() types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, state := range ps.ReplicaStates {
			rState := state.(*RedisNodeState)
			filteredLog := filterLogCommit(rState.Logs, rState.Commit)
			for _, entry := range filteredLog {
				if EntryTypeToString(entry.Type) == "NORMAL" {
					return true
				}
			}
		}
		return false
	}
}

// returns true if there is at least one committed NORMAL entry in the specified term in any of the logs.
func AtLeastOneCommittedNormalEntryInTerm(term int) types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, state := range ps.ReplicaStates {
			rState := state.(*RedisNodeState)
			filteredLog := filterLogCommit(rState.Logs, rState.Commit)
			for _, entry := range filteredLog {
				if EntryTypeToString(entry.Type) == "NORMAL" && entry.Term == term {
					return true
				}
			}
		}
		return false
	}
}

// returns true if all the replicas are in the specified term and either in leader or follower state
func OnlyFollowersAndLeaderInTerm(term int) types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, state := range ps.ReplicaStates {
			repState := state.(*RedisNodeState)
			if repState.State != "leader" && repState.State != "follower" || repState.Term != term {
				return false
			}
		}
		return true
	}
}

// returns true if there is the specified gap in the committed logs of two processes, with both logs having at least one entry
func DiffInCommittedEntries(val int) types.RewardFuncSingle {
	return func(s types.State) bool {
		minLength := math.MaxInt
		maxLength := 0
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, state := range ps.ReplicaStates {
			rState := state.(*RedisNodeState)
			filteredLog := filterLogCommit(rState.Logs, rState.Commit)
			if len(filteredLog) > maxLength {
				maxLength = len(filteredLog)
			}
			if len(filteredLog) < minLength {
				minLength = len(filteredLog)
			}
		}
		return (maxLength-minLength) >= val && maxLength > 0 && minLength > 0
	}
}

// returns true if there is the specified gap in the logs of two processes, with both logs having at least one entry
func DiffInEntries(val int) types.RewardFuncSingle {
	return func(s types.State) bool {
		minLength := math.MaxInt
		maxLength := 0
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, state := range ps.ReplicaStates {
			rState := state.(*RedisNodeState)
			if len(rState.Logs) > maxLength {
				maxLength = len(rState.Logs)
			}
			if len(rState.Logs) < minLength {
				minLength = len(rState.Logs)
			}
		}
		return (maxLength-minLength) >= val && maxLength > 0 && minLength > 0
	}
}

// returns true if there are at least the specified number of pending requests
func PendingRequestsAtLeast(val int) types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		return len(ps.PendingRequests) >= val
	}
}
