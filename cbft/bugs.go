package cbft

import "github.com/zeu5/raft-rl-test/types"

func ReachedRound1() func(*types.Trace) (bool, int) {
	return func(trace *types.Trace) (bool, int) {
		for i := 0; i < trace.Len(); i++ {
			s, _, _, _ := trace.Get(i)
			for _, rs := range s.(*types.Partition).ReplicaStates {
				cs := rs.(*CometNodeState)
				if cs.Round >= 1 {
					return true, i
				}
			}
		}
		return false, -1
	}
}

func DifferentProposers() func(*types.Trace) (bool, int) {
	return func(trace *types.Trace) (bool, int) {
		for i := 0; i < trace.Len(); i++ {
			s, _, _, _ := trace.Get(i)
			proposers := make(map[string]bool)
			for _, rs := range s.(*types.Partition).ReplicaStates {
				cs := rs.(*CometNodeState)
				if cs.Proposer != nil {
					proposers[string(cs.Proposer.Address)] = true
				}
			}
			if len(proposers) > 1 {
				return true, i
			}
		}
		return false, -1
	}
}
