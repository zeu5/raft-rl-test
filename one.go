package main

import (
	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/raft"
	"github.com/zeu5/raft-rl-test/types"
)

func One(episodes, horizon int, saveFile string) {
	raftConfig := raft.RaftEnvironmentConfig{
		Replicas:      3,
		ElectionTick:  10,
		HeartbeatTick: 1,
		Timeouts:      true,
	}
	c := types.NewComparison(raft.RaftAnalyzer, raft.RaftPlotComparator(saveFile))
	c.AddExperiment(types.NewExperiment("RL", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      types.NewSoftMaxNegPolicy(0.3, 0.7),
		Environment: raft.NewRaftEnvironment(raftConfig),
	}))
	c.AddExperiment(types.NewExperiment("Random", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      types.NewRandomPolicy(),
		Environment: raft.NewRaftEnvironment(raftConfig),
	}))
	raftConfig.Timeouts = false
	c.AddExperiment(types.NewExperiment("RL-NoTimeouts", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      types.NewSoftMaxNegPolicy(0.3, 0.7),
		Environment: raft.NewRaftEnvironment(raftConfig),
	}))
	c.AddExperiment(types.NewExperiment("Random-NoTimeouts", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      types.NewRandomPolicy(),
		Environment: raft.NewRaftEnvironment(raftConfig),
	}))

	c.Run()
}

func OneCommand() *cobra.Command {
	return &cobra.Command{
		Use: "one",
		Run: func(cmd *cobra.Command, args []string) {
			One(episodes, horizon, saveFile)
		},
	}
}
