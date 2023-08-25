package rsl

import "github.com/zeu5/raft-rl-test/types"

func Decided() types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, rs := range pS.ReplicaStates {
			decided := rs.(LocalState).decided
			if decided > 0 {
				return true
			}
		}
		return false
	}
}
