package raft

import (
	"github.com/zeu5/raft-rl-test/types"
	"go.etcd.io/raft/v3"
)

func LeaderElected() *types.Monitor {
	monitor := types.NewMonitor()
	builder := monitor.Build()
	// builder.On(HasLeader().Not(), types.InitState)
	builder.On(HasLeader(), "LeaderElected").MarkSuccess()
	return monitor
}

func HasLeader() types.MonitorCondition {
	return func(s types.State, _ types.Action, _ types.State) bool {
		nodeStates, ok := getNodeStates(s)
		if !ok {
			return false
		}

		for _, s := range nodeStates {
			if s.RaftState == raft.StateLeader {
				return true
			}
		}
		return false
	}
}

func getNodeStates(state types.State) (map[uint64]raft.Status, bool) {
	raftState, ok := state.(RaftStateType)
	if !ok {
		return nil, false
	}
	return raftState.GetNodeStates(), true
}

func LeaderApplied() types.MonitorCondition {
	return func(s types.State, _ types.Action, ns types.State) bool {
		rS1, ok := getNodeStates(s)
		if !ok {
			return false
		}
		rS2, ok := getNodeStates(ns)
		if !ok {
			return false
		}
		var leader uint64 = 0
		for node, s := range rS1 {
			if s.RaftState == raft.StateLeader {
				leader = node
				break
			}
		}
		if leader == 0 {
			return false
		}
		first := rS1[leader].Applied
		second := rS2[leader].Applied
		return second > first
	}
}

func LeaderCommittedRequest() *types.Monitor {
	monitor := types.NewMonitor()
	builder := monitor.Build()
	builder.On(LeaderApplied(), "LeaderCommittedRequest").MarkSuccess()
	return monitor
}
