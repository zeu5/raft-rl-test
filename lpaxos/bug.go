package lpaxos

import (
	"fmt"

	"github.com/zeu5/raft-rl-test/types"
)

func SafetyBug() func(*types.Trace) bool {
	return func(t *types.Trace) bool {
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
						return true
					}
					processStates[process] = curState
				}
			}
		}
		return false
	}
}

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

func BugAnalyzer(bug func(*types.Trace) bool) types.Analyzer {
	return func(run int, s string, traces []*types.Trace) types.DataSet {
		occurrences := 0
		for _, t := range traces {
			if bug(t) {
				occurrences += 1
			}
		}
		return occurrences
	}
}

func BugComparator() types.Comparator {
	return func(run int, s []string, ds []types.DataSet) {
		for i, exp := range s {
			bugOccurrences := ds[i].(int)
			fmt.Printf("For run:%d, experiment: %s, bug occurrences: %d\n", run, exp, bugOccurrences)
		}
	}
}
