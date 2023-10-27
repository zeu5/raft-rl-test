package lpaxos

import "github.com/zeu5/raft-rl-test/types"

// defining predicates over states

func InStep(step Step) types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if ok {
			for _, c := range pS.ReplicaStates {
				lC := c.(LNodeState)
				if ok && int(lC.Step) == int(step) {
					return true
				}
			}
		} else {
			lS, ok := s.(*LPaxosState)
			if !ok {
				return false
			}
			for _, c := range lS.NodeStates {
				if c.Step == step {
					return true
				}
			}
		}
		return false
	}
}

func InPhase(phase int) types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if ok {
			for _, c := range pS.ReplicaStates {
				lC := c.(LNodeState)
				if ok && lC.Phase == phase {
					return true
				}
			}
		} else {
			lS, ok := s.(*LPaxosState)
			if !ok {
				return false
			}
			for _, c := range lS.NodeStates {
				if c.Phase == phase {
					return true
				}
			}
		}
		return false
	}
}

func Commit() types.RewardFunc {
	return func(s types.State, ns types.State) bool {
		pS, ok := s.(*types.Partition)
		npS, _ := ns.(*types.Partition)
		if ok {
			for id, c := range pS.ReplicaStates {
				log := c.(LNodeState).Log
				nextLog := npS.ReplicaStates[id].(LNodeState).Log
				if nextLog.Size() > log.Size() {
					return true
				}
			}
		} else {
			lS, ok := s.(*LPaxosState)
			nlS, _ := ns.(*LPaxosState)
			if !ok {
				return false
			}
			for id, c := range lS.NodeStates {
				nextLog := nlS.NodeStates[id].Log
				if nextLog.Size() > c.Log.Size() {
					return true
				}
			}
		}
		return false
	}
}

func Decided() types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if !ok {
			return false
		}
		for _, rs := range pS.ReplicaStates {
			decided := rs.(LNodeState).Decided
			if decided > 0 {
				return true
			}
		}
		return false
	}
}

func OnlyMajorityDecidedOne() types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if ok {
			numReplicas := len(pS.ReplicaColors)
			numDecided := 0
			for _, rs := range pS.ReplicaStates {
				decided := rs.(LNodeState).Decided
				if decided > 0 {
					numDecided += 1
				}
			}
			return numDecided == numReplicas/2+1
		}
		return false
	}
}

func EmptyLogLeader() types.RewardFuncSingle {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if ok {
			for id, rs := range pS.ReplicaStates {
				lS := rs.(LNodeState)
				if lS.Leader == id && lS.Log.Size() == 0 {
					return true
				}
			}
		}
		return false
	}
}
