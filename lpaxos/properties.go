package lpaxos

import "github.com/zeu5/raft-rl-test/types"

// TODO: define some basic properties to bias exploration
// Benchmark between different environments for same algorithm
// Basic property - check if a process moved to propose phase

func InPhase(step Step) types.RewardFunc {
	return func(s types.State, _ types.State) bool {
		pS, ok := s.(*types.Partition)
		// nS, _ := ns.(*types.Partition)
		if ok {
			for _, c := range pS.ReplicaColors {
				if int(c) == int(step) {
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

func Commit() types.RewardFunc {
	return func(s types.State, ns types.State) bool {
		pS, ok := s.(*types.Partition)
		// npS, _ := s.(*types.Partition)
		if ok {
			for _, c := range pS.ReplicaStates {
				log := c.(LNodeState).Log
				if log.Size() > 0 {
					return true
				}
				// nextLog := npS.ReplicaStates[id].(LNodeState).Log
				// if nextLog.Size() != log.Size() {
				// 	return true
				// }
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
