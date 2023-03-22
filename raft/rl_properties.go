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
	return func(trace *types.Trace) bool {
		last, _, _, ok := trace.Last()
		if !ok {
			return false
		}
		raftState, ok := last.(*RaftState)
		if !ok {
			return false
		}
		for _, s := range raftState.NodeStates {
			if s.RaftState == raft.StateLeader {
				return true
			}
		}
		return false
	}
}

func LeaderApplied() types.MonitorCondition {
	return func(trace *types.Trace) bool {
		s1, _, s2, ok := trace.Last()
		if !ok {
			return false
		}
		rS1 := s1.(*RaftState)
		rS2 := s2.(*RaftState)
		var leader uint64 = 0
		for node, s := range rS1.NodeStates {
			if s.RaftState == raft.StateLeader {
				leader = node
				break
			}
		}
		if leader == 0 {
			return false
		}
		first := rS1.NodeStates[leader].Applied
		second := rS2.NodeStates[leader].Applied
		return second > first
	}
}

func LeaderCommittedRequest() *types.Monitor {
	monitor := types.NewMonitor()
	builder := monitor.Build()
	builder.On(LeaderApplied(), "LeaderCommittedRequest").MarkSuccess()
	return monitor
}
