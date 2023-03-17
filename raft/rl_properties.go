package raft

import (
	"github.com/zeu5/raft-rl-test/rl"
	"go.etcd.io/raft/v3"
)

func LeaderElected() *rl.Monitor {
	monitor := rl.NewMonitor()
	builder := monitor.Build()
	// builder.On(HasLeader().Not(), rl.InitState)
	builder.On(HasLeader(), "LeaderElected").MarkSuccess()
	return monitor
}

func HasLeader() rl.MonitorCondition {
	return func(trace *rl.Trace) bool {
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
