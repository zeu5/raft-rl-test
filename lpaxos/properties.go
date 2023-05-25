package lpaxos

import "github.com/zeu5/raft-rl-test/types"

// TODO: define some basic properties to bias exploration
// Benchmark between different environments for same algorithm
// Basic property - check if a process moved to propose phase

func InProposePhase() types.RewardFunc {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if ok {
			for _, c := range pS.ReplicaColors {
				if int(c) == int(StepPromise) {
					return true
				}
			}
		} else {
			lS, ok := s.(*LPaxosState)
			if !ok {
				return false
			}
			for _, c := range lS.NodeStates {
				if c.Step == StepPromise {
					return true
				}
			}
		}
		return false
	}
}

func InPreparePhase() types.RewardFunc {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if ok {
			for _, c := range pS.ReplicaColors {
				if int(c) == int(StepPrepare) {
					return true
				}
			}
		} else {
			lS, ok := s.(*LPaxosState)
			if !ok {
				return false
			}
			for _, c := range lS.NodeStates {
				if c.Step == StepPrepare {
					return true
				}
			}
		}
		return false
	}
}

func Commit() types.RewardFunc {
	return func(s types.State) bool {
		pS, ok := s.(*types.Partition)
		if ok {
			for _, c := range pS.ReplicaStates {
				log := c.(LNodeState).Log
				if log.Size() > 0 {
					return true
				}
			}
		} else {
			lS, ok := s.(*LPaxosState)
			if !ok {
				return false
			}
			for _, c := range lS.NodeStates {
				if c.Log.Size() > 0 {
					return true
				}
			}
		}
		return false
	}
}
