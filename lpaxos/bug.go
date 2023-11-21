package lpaxos

import (
	"github.com/zeu5/raft-rl-test/types"
)

// this is a function that checks for a certain type of bug in a trace
func SafetyBug() func(*types.Trace) (bool, int) {
	return func(t *types.Trace) (bool, int) {
		processStates := make(map[uint64]LNodeState)
		for i := 0; i < t.Len(); i++ {
			s, _, _, _ := t.Get(i)
			pS, ok := s.(*types.Partition)
			if ok {
				for process, state := range pS.ReplicaStates {
					curState := state.(LNodeState)
					if _, ok := processStates[process]; !ok {
						processStates[process] = curState
					}
					prevState := processStates[process]
					if curState.Decided < prevState.Decided || !isLogPrefix(getLogPrefix(prevState.Log, prevState.Decided), getLogPrefix(curState.Log, prevState.Decided)) {
						// At any point the old log is not a prefix of the new log then we have a bug (we only care about the decided prefix)
						return true, i
					}
					processStates[process] = curState
				}
			}
		}
		return false, -1
	}
}

// other functions / auxiliary
func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}

func getLogPrefix(l *Log, upto int) []Entry {
	entries := make([]Entry, upto)
	for i, e := range l.Entries() {
		if i < upto {
			entries[i] = e
		}
	}
	return entries
}

func isLogPrefix(l1, l2 []Entry) bool {
	if len(l1) < len(l2) {
		return false
	}

	for i, e := range l1 {
		e2 := l2[i]
		if !e2.Eq(e) {
			return false
		}
	}
	return true
}
