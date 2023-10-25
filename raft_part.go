package main

import (
	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/raft"
	"github.com/zeu5/raft-rl-test/types"
)

func RaftPart(episodes, horizon int, saveFile string) {
	raftConfig := raft.RaftEnvironmentConfig{
		Replicas:      3,
		ElectionTick:  10,
		HeartbeatTick: 3,
		Timeouts:      timeouts,
		Requests:      requests,
	}

	colors := []raft.RaftColorFunc{raft.ColorState(), raft.ColorCommit(), raft.ColorLeader(), raft.ColorVote(), raft.ColorBoundedTerm(5)}

	c := types.NewComparison(raft.RaftAnalyzer(saveFile, colors...), raft.RaftPlotComparator(saveFile), runs)
	c.AddExperiment(types.NewExperiment("RL", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      policies.NewSoftMaxNegFreqPolicy(0.3, 0.7, 1),
		Environment: getRaftPartEnv(raftConfig, colors),
	}))
	c.AddExperiment(types.NewExperiment("Random", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      types.NewRandomPolicy(),
		Environment: getRaftPartEnv(raftConfig, colors),
	}))
	c.AddExperiment(types.NewExperiment("BonusMaxRL", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      policies.NewBonusPolicyGreedy(0.1, 0.99, 0.2),
		Environment: getRaftPartEnv(raftConfig, colors),
	}))

	c.Run()
}

func getRaftPartEnv(config raft.RaftEnvironmentConfig, colors []raft.RaftColorFunc) types.Environment {

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
			RaftPart(episodes, horizon, saveFile)
		},
	}
	cmd.PersistentFlags().IntVarP(&requests, "requests", "r", 1, "Number of requests to run with")
	return cmd
}
