package lpaxos

import "github.com/zeu5/raft-rl-test/types"

func InconsistentLogsCond() types.MonitorCondition {
	return func(s types.State, _ types.Action, _ types.State) bool {
		ls, ok := s.(*LPaxosState)
		if !ok {
			return false
		}
		maxDecided := 0
		decidedLogs := make(map[uint64][]Entry)
		for node, s := range ls.NodeStates {
			decidedLogs[node] = make([]Entry, 0)
			if s.Decided > 0 {
				upto := s.Decided
				if s.Log.Size() < upto {
					upto = s.Log.Size()
				}
				if upto > maxDecided {
					maxDecided = upto
				}
				decidedLogs[node] = s.Log.Copy().Entries()[0:upto]
			}
		}
		for _, l := range decidedLogs {
			if len(l) != maxDecided {
				return true
			}
		}
		for i := 0; i < maxDecided; i++ {
			entries := make([]Entry, len(decidedLogs))
			for _, l := range decidedLogs {
				entries = append(entries, l[i])
			}
			for i := 1; i < len(entries); i++ {
				if !entries[i].Eq(entries[0]) {
					return true
				}
			}
		}
		return false
	}
}

func InconsistentLogs() *types.Monitor {
	m := types.NewMonitor()
	m.Build().On(InconsistentLogsCond(), "InconsistentLog").MarkSuccess()
	return m
}
