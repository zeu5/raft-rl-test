package main

import (
	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/raft"
	"github.com/zeu5/raft-rl-test/types"
)

func Three(episodes, horizon int, saveFile string) {
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
	c.AddExperiment(types.NewExperiment("BonusRL", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      policies.NewBonusPolicyGreedy(0.3, 0.99, false),
		Environment: raft.NewRaftEnvironment(raftConfig),
	}))
	c.AddExperiment(types.NewExperiment("BonusMaxRL", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      policies.NewBonusPolicyGreedy(0.3, 0.99, true),
		Environment: raft.NewRaftEnvironment(raftConfig),
	}))

	c.Run()
}

func ThreeCommand() *cobra.Command {
	return &cobra.Command{
		Use: "three",
		Run: func(cmd *cobra.Command, args []string) {
			Three(episodes, horizon, saveFile)
		},
	}
}
