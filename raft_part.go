package main

import (
	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/raft"
	"github.com/zeu5/raft-rl-test/types"
)

func RaftPart(episodes, horizon int, saveFile string) {
	raftConfig := raft.RaftEnvironmentConfig{
		Replicas:      5,
		ElectionTick:  15,
		HeartbeatTick: 3,
		Timeouts:      timeouts,
		Requests:      requests,
	}
	c := types.NewComparison(raft.RaftAnalyzer(saveFile), raft.RaftPlotComparator(saveFile), runs)
	c.AddExperiment(types.NewExperiment("RL", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      types.NewSoftMaxNegPolicy(0.3, 0.7, 1),
		Environment: getRaftPartEnv(raftConfig),
	}))
	c.AddExperiment(types.NewExperiment("Random", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      types.NewRandomPolicy(),
		Environment: getRaftPartEnv(raftConfig),
	}))
	c.AddExperiment(types.NewExperiment("BonusMaxRL", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      policies.NewBonusPolicyGreedy(0.1, 0.99, 0),
		Environment: getRaftPartEnv(raftConfig),
	}))

	c.Run()
}

func getRaftPartEnv(config raft.RaftEnvironmentConfig) types.Environment {
	colors := []raft.RaftColorFunc{raft.ColorState(), raft.ColorCommit(), raft.ColorLeader(), raft.ColorVote()}

	return types.NewPartitionEnv(types.PartitionEnvConfig{
		Painter:                raft.NewRaftStatePainter(colors...),
		Env:                    raft.NewPartitionEnvironment(config),
		TicketBetweenPartition: 3,
		MaxMessagesPerTick:     3,
		StaySameStateUpto:      2,
		NumReplicas:            config.Replicas,
	})
}

func RaftPartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "raft-part",
		Run: func(cmd *cobra.Command, args []string) {
			Raft(episodes, horizon, saveFile)
		},
	}
	cmd.PersistentFlags().IntVarP(&requests, "requests", "r", 1, "Number of requests to run with")
	return cmd
}
