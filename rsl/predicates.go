package rsl

import "github.com/zeu5/raft-rl-test/types"

func Decided() types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, rs := range pS.ReplicaStates {
			decided := rs.(LocalState).Decided
			if decided > 0 {
				return true
			}
		}
		return false
	}
}

func InState(state RSLState) types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, rs := range ps.ReplicaStates {
			if rs.(LocalState).State == state {
				return true
			}
		}
		return false
	}
}

func NodePrimary(node uint64) types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for n, rs := range ps.ReplicaStates {
			if rs.(LocalState).State == StateStablePrimary && n == node {
				return true
			}
		}
		return false
	}
}

func NodeDecided(node uint64) types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for n, rs := range pS.ReplicaStates {
			decided := rs.(LocalState).Decided
			if n == node && decided > 0 {
				return true
			}
		}
		return false
	}
}

func NumDecided(d int) types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, rs := range pS.ReplicaStates {
			decided := rs.(LocalState).Decided
			if decided == d {
				return true
			}
		}
		return false
	}
}

func NodeNumDecided(node uint64, d int) types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for n, rs := range pS.ReplicaStates {
			decided := rs.(LocalState).Decided
			if decided == d && node == n {
				return true
			}
		}
		return false
	}
}

func InBallot(ballot int) types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, rs := range ps.ReplicaStates {
			if rs.(LocalState).MaxAcceptedProposal.Ballot.Num == ballot {
				return true
			}
		}
		return false
	}
}

func NodeInBallot(node uint64, ballot int) types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for n, rs := range ps.ReplicaStates {
			if rs.(LocalState).MaxAcceptedProposal.Ballot.Num == ballot && n == node {
				return true
			}
		}
		return false
	}
}

func InPreparedBallot(ballot int) types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, rs := range ps.ReplicaStates {
			if rs.(LocalState).MaxPreparedBallot.Num == ballot {
				return true
			}
		}
		return false
	}
}

func NodeInPreparedBallot(node uint64, ballot int) types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for n, rs := range ps.ReplicaStates {
			if rs.(LocalState).MaxPreparedBallot.Num == ballot && n == node {
				return true
			}
		}
		return false
	}
}
