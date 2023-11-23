package rsl

import "github.com/zeu5/raft-rl-test/types"

func InconsistentLogState(state *types.Partition) bool {
	maxLogLength := 0
	decidedLogs := make(map[uint64][]Command)
	for node, rs := range state.ReplicaStates {
		ls := rs.(LocalState)
		decidedLogs[node] = make([]Command, len(ls.Log.Decided))
		for i, c := range ls.Log.Decided {
			decidedLogs[node][i] = c.Copy()
		}
		if ls.Log.NumDecided() > maxLogLength {
			maxLogLength = ls.Log.NumDecided()
		}
	}
	for j := 0; j < maxLogLength; j++ {
		commands := make(map[string]Command)
		for _, decided := range decidedLogs {
			if j < len(decided) {
				commands[decided[j].Hash()] = decided[j].Copy()
			}
		}
		if len(commands) > 1 {
			return true
		}
	}
	return false
}

func InconsistentLogs() func(*types.Trace) (bool, int) {
	return func(t *types.Trace) (bool, int) {
		for i := 0; i < t.Len(); i++ {
			s, _, _, _ := t.Get(i)
			ps, ok := s.(*types.Partition)
			if !ok {
				continue
			}
			if InconsistentLogState(ps) {
				return true, i
			}
		}

		return false, -1
	}
}

func MultiplePrimaries() func(*types.Trace) (bool, int) {
	return func(t *types.Trace) (bool, int) {
		for i := 0; i < t.Len(); i++ {
			s, _, _, _ := t.Get(i)
			ps, ok := s.(*types.Partition)
			if !ok {
				continue
			}
			primaries := make(map[int]uint64)
			for node, rs := range ps.ReplicaStates {
				ls := rs.(LocalState)
				if ls.State == StateStablePrimary {
					ballot := ls.MaxAcceptedProposal.Ballot.Num
					if _, ok := primaries[ballot]; ok {
						// There is already a primary for this ballot number
						return true, i
					}
					primaries[ballot] = node
				}
			}
		}

		return false, -1
	}
}
