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

func NodePrimaryInBallot(node uint64, ballot int) types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for n, rs := range ps.ReplicaStates {
			ls := rs.(LocalState)
			if ls.State == StateStablePrimary && n == node && ls.MaxAcceptedProposal.Ballot.Num == ballot {
				return true
			}
		}
		return false
	}
}

func InStateAndBallot(state RSLState, ballot int) types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, rs := range ps.ReplicaStates {
			ls := rs.(LocalState)
			if ls.State == state && ls.MaxAcceptedProposal.Ballot.Num == ballot {
				return true
			}
		}
		return false
	}
}

func OutSyncBallotBy(diff int) types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		ballotsMap := make(map[int]bool)
		for _, rs := range ps.ReplicaStates {
			ls := rs.(LocalState)
			ballotsMap[ls.MaxAcceptedProposal.Ballot.Num] = true
		}
		ballots := make([]int, 0)
		for b := range ballotsMap {
			ballots = append(ballots, b)
		}

		maxBallot := ballots[0]
		minBallot := ballots[0]
		for _, b := range ballots {
			if b > maxBallot {
				maxBallot = b
			}
			if b < minBallot {
				minBallot = b
			}
		}
		return maxBallot-minBallot >= diff
	}
}

func AllInSync() types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		ballotsMap := make(map[int]bool)
		for _, rs := range ps.ReplicaStates {
			ls := rs.(LocalState)
			ballotsMap[ls.MaxAcceptedProposal.Ballot.Num] = true
		}
		if len(ballotsMap) != 1 {
			return false
		}
		for b := range ballotsMap {
			return b != 0
		}
		return false
	}
}

func AllAtMostBallot(maxBallot int) types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		ballotsMap := make(map[int]bool)
		for _, rs := range ps.ReplicaStates {
			ls := rs.(LocalState)
			ballotsMap[ls.MaxAcceptedProposal.Ballot.Num] = true
		}
		for b := range ballotsMap {
			if b > maxBallot {
				return false
			}
		}
		return true
	}
}

func AllAtLeastBallot(minBallot int) types.RewardFuncSingle {
	return func(s types.State) bool {
		ps, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		ballotsMap := make(map[int]bool)
		for _, rs := range ps.ReplicaStates {
			ls := rs.(LocalState)
			ballotsMap[ls.MaxAcceptedProposal.Ballot.Num] = true
		}
		for b := range ballotsMap {
			if b < minBallot {
				return false
			}
		}
		return true
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

func AtMostDecided(d int) types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, rs := range pS.ReplicaStates {
			decided := rs.(LocalState).Decided
			if decided > d {
				return false
			}
		}
		return true
	}
}

func AtLeastDecided(d int) types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, rs := range pS.ReplicaStates {
			decided := rs.(LocalState).Decided
			if decided < d {
				return false
			}
		}
		return true
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
