package main

import (
	"os"

	"github.com/zeu5/raft-rl-test/raft"
	"github.com/zeu5/raft-rl-test/types"
)

func One() {
	c := types.NewComparison(raft.RaftAnalyzer, raft.RaftPlotComparator(os.Args[1]))
	c.AddExperiment(types.NewExperiment("RL", &types.AgentConfig{
		Episodes: 100,
		Horizon:  20,
		Policy:   types.NewSoftMaxNegPolicy(0.3, 0.7),
		Environment: raft.NewRaftEnvironment(raft.RaftEnvironmentConfig{
			Replicas:      3,
			ElectionTick:  10,
			HeartbeatTick: 1,
		}),
	}))
	c.AddExperiment(types.NewExperiment("Random", &types.AgentConfig{
		Episodes: 100,
		Horizon:  20,
		Policy:   types.NewRandomPolicy(),
		Environment: raft.NewRaftEnvironment(raft.RaftEnvironmentConfig{
			Replicas:      3,
			ElectionTick:  10,
			HeartbeatTick: 1,
		}),
	}))

	c.Run()
}
