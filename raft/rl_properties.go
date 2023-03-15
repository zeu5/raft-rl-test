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
		last, _, _, _ := trace.Get(trace.Len() - 1)
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
