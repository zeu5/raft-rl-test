package main

import (
	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/raft"
	"github.com/zeu5/raft-rl-test/types"
)

func Four(episodes, horizon int, saveFile string) {
	raftConfig := raft.RaftEnvironmentConfig{
		Replicas:      3,
		ElectionTick:  10,
		HeartbeatTick: 1,
		Timeouts:      true,
	}

	c := types.NewComparison(raft.RaftAnalyzer, raft.RaftPlotComparator(saveFile, raft.DefaultFilter()))
	c.AddExperiment(types.NewExperiment("AbstractRL", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      types.NewSoftMaxNegPolicy(0.3, 0.7),
		Environment: raft.NewLinkRaftEnvironment(raftConfig),
	}))

	c.AddExperiment(types.NewExperiment("RL", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      types.NewSoftMaxNegPolicy(0.3, 0.7),
		Environment: raft.NewRaftEnvironment(raftConfig),
	}))

	c.AddExperiment(types.NewExperiment("AbstractRandom", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      types.NewRandomPolicy(),
		Environment: raft.NewLinkRaftEnvironment(raftConfig),
	}))

	c.AddExperiment(types.NewExperiment("Random", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      types.NewRandomPolicy(),
		Environment: raft.NewRaftEnvironment(raftConfig),
	}))

	c.Run()
}

func FourCommand() *cobra.Command {
	return &cobra.Command{
		Use: "four",
		Run: func(cmd *cobra.Command, args []string) {
			Four(episodes, horizon, saveFile)
		},
	}
}
