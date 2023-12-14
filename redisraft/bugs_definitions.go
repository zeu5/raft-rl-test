package redisraft

import (
	"math"

	"github.com/zeu5/raft-rl-test/types"
)

// checks if the log size of a replica decreases throughout an execution
func ReducedLog() func(*types.Trace) (bool, int) {
	return func(t *types.Trace) (bool, int) {
		replicasLogs := make(map[uint64][]RedisEntry) // map of processID : Log (list of entries)

		for i := 0; i < t.Len(); i++ { // foreach (state, action, new_state, reward) in the trace
			s, _, _, _ := t.Get(i) // take state s
			pS, ok := s.(*types.Partition)
			if ok {
				for replica_id, elem := range pS.ReplicaStates {
					repState := elem.(*RedisNodeState)
					curLog := repState.Logs
					commitIndex := repState.Commit - 1 // apparently need to subtract 1...

					if _, ok := replicasLogs[replica_id]; !ok { // init empty list if previous replica log is not present
						replicasLogs[replica_id] = make([]RedisEntry, 0)
					}

					commitLog := filterLogCommit(curLog, commitIndex)

					if len(commitLog) < len(replicasLogs[replica_id]) && (repState.Term > 0) { // check if log size decreased
						return true, i // BUG FOUND
					}

					replicasLogs[replica_id] = copyLog(commitLog) // update previous state log with the current one for next iteration
				}
			}
		}
		return false, -1
	}
}

// checks if a committed entry of a replica has been changed throughout an execution
func ModifiedLog() func(*types.Trace) (bool, int) {
	return func(t *types.Trace) (bool, int) {
		replicasLogs := make(map[uint64][]RedisEntry) // map of processID : Log (list of entries)

		for i := 0; i < t.Len(); i++ { // foreach (state, action, new_state, reward) in the trace
			s, _, _, _ := t.Get(i) // take state s
			pS, ok := s.(*types.Partition)
			if ok {
				for replica_id, elem := range pS.ReplicaStates {
					repState := elem.(*RedisNodeState)
					curLog := repState.Logs
					commitIndex := repState.Commit - 1 // apparently need to subtract 1...

					if _, ok := replicasLogs[replica_id]; !ok { // init empty list if previous replica log is not present
						replicasLogs[replica_id] = make([]RedisEntry, 0)
					}

					commitLog := filterLogCommit(curLog, commitIndex) // take only entries with index at most equal to commit
					commitLog = sortLogIndex(commitLog)               // sort them by index (to compare them)

					for j := 0; j < min(len(replicasLogs[replica_id]), len(commitLog)); j++ { // for the size of the old log (ignore newly appended entries)
						if !eqEntry(commitLog[j], replicasLogs[replica_id][j]) { // check if they are equal
							return true, i // BUG FOUND
						}
					}

					replicasLogs[replica_id] = copyLog(curLog) // update previous state log with the current one for next iteration
				}
			}
		}
		return false, -1
	}
}

// checks if any replica has an inconsistent log w.r.t. other replicas
func InconsistentLogs() func(*types.Trace) (bool, int) {
	return func(t *types.Trace) (bool, int) {
		for i := 0; i < t.Len(); i++ { // foreach (state, action, new_state, reward) in the trace
			s, _, _, _ := t.Get(i) // take state s
			pS, ok := s.(*types.Partition)
			if ok {
				// make a list of logs, starting at index 0
				logsList := make([][]RedisEntry, 0, len(pS.ReplicaStates))
				for _, value := range pS.ReplicaStates { // build all replicas committed logs, sorted by index
					repState := value.(*RedisNodeState)
					curLog := repState.Logs
					commitIndex := repState.Commit - 1                // apparently need to subtract 1...
					commitLog := filterLogCommit(curLog, commitIndex) // take only entries with index at most equal to commit
					commitLog = sortLogIndex(commitLog)               // sort them by index (to compare them)
					logsList = append(logsList, commitLog)
				}

				for j1 := 0; j1 < len(logsList); j1++ { // for each replica
					for j2 := j1; j2 < len(logsList); j2++ { // for each other replica
						minSize := min(len(logsList[j1]), len(logsList[j2])) // take the minimum length among the two logs

						for k := 0; k < minSize; k++ { // for each entry
							if !eqEntryNoID(logsList[j1][k], logsList[j2][k]) { // check if they are equal
								return true, i // BUG FOUND
							}
						}
					}
				}
			}
		}
		return false, -1
	}
}

// checks if any replica has an inconsistent log w.r.t. other replicas
func TruePredicate() func(*types.Trace) (bool, int) {
	return func(t *types.Trace) (bool, int) {
		return true, 0
	}
}

// checks if a replica committed two NORMAL entries in different terms
func EntriesInDifferentTermsDummy() func(*types.Trace) (bool, int) {
	return func(t *types.Trace) (bool, int) {
		for i := 0; i < t.Len(); i++ { // foreach (state, action, new_state, reward) in the trace
			s, _, _, _ := t.Get(i) // take state s
			pS, ok := s.(*types.Partition)
			if ok {
				for _, elem := range pS.ReplicaStates {
					repState := elem.(*RedisNodeState)
					curLog := repState.Logs
					commitIndex := repState.Commit - 1 // apparently need to subtract 1...

					commitLog := filterLogCommit(curLog, commitIndex)

					terms := make(map[int]bool)
					for _, ent := range commitLog {
						if EntryTypeToString(ent.Type) == "NORMAL" {
							terms[ent.Term] = true
						}
					}

					if len(terms) > 1 {
						return true, i
					}
				}
			}
		}
		return false, -1
	}
}

// take a log and a commit index value, and discard all entries with index higher than the commit index
func filterLogCommit(log []RedisEntry, commitIndex int) []RedisEntry {
	result := make([]RedisEntry, 0)

	for _, entry := range log {
		if entry.Index <= commitIndex {
			result = append(result, entry.Copy())
		}
	}

	return result
}

func eqEntry(a RedisEntry, b RedisEntry) bool {
	return a.DataLen == b.DataLen && a.ID == b.ID && a.Term == b.Term
}

func eqEntryNoID(a RedisEntry, b RedisEntry) bool {
	return a.DataLen == b.DataLen && a.Term == b.Term
}

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}

// sort a log by increasing index value for entries
func sortLogIndex(log []RedisEntry) []RedisEntry {
	init := copyLog(log) // remaining entries
	result := make([]RedisEntry, 0)
	size := len(init)

	for i := 0; i < size; i++ {
		min := math.MaxInt32
		index := -1

		for j, entry := range init { // find min index in remaining entries
			if entry.Index < min {
				index = j
				min = entry.Index
			}
		}

		result = append(result, init[index].Copy()) // append it to the result
		init = removeEntry(init, index)             // remove it from the remaining entries
	}

	return result

}

// remove element at index i from a slice, without preserving order (efficient)
func removeEntry(s []RedisEntry, i int) []RedisEntry {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}
