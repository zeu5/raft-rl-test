package lpaxos

import (
	"fmt"

	"github.com/zeu5/raft-rl-test/types"
)

func SafetyBug() func(*types.Trace) bool {
	return func(t *types.Trace) bool {
		processLogs := make(map[uint64]*Log)
		for i := 0; i < t.Len(); i++ {
			s, _, _, _ := t.Get(i)
			pS, ok := s.(*types.Partition)
			if ok {
				for process, state := range pS.ReplicaStates {
					lS := state.(LNodeState)
					if _, ok := processLogs[process]; !ok {
						processLogs[process] = lS.Log
					}
					if !isLogPrefix(processLogs[process], lS.Log) {
						// At any point the old log is not a prefix of the new log then we have a bug
						return true
					}
					processLogs[process] = lS.Log
				}
			}
		}
		return false
	}
}

func isLogPrefix(l1, l2 *Log) bool {
	if l1.Size() > l2.Size() {
		return false
	}
	for i, e := range l1.Entries() {
		e2, ok := l2.Get(i)
		if !ok || !e2.Eq(e) {
			return false
		}
	}
	return true
}

func BugAnalyzer(bug func(*types.Trace) bool) types.Analyzer {
	return func(s string, traces []*types.Trace) types.DataSet {
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
	return func(s []string, ds []types.DataSet) {
		for i, exp := range s {
			bugOccurrences := ds[i].(int)
			fmt.Printf("For experiment: %s, bug occurrences: %d\n", exp, bugOccurrences)
		}
	}
}
