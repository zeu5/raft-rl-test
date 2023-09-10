package main

import (
	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/rsl"
	"github.com/zeu5/raft-rl-test/types"
)

func RSLExploration() {
	config := rsl.RSLEnvConfig{
		Nodes: 3,
		NodeConfig: rsl.NodeConfig{
			HeartBeatInterval:       2,
			NoProgressTimeout:       15,
			BaseElectionDelay:       10,
			InitializeRetryInterval: 5,
			NewLeaderGracePeriod:    15,
			VoteRetryInterval:       5,
			PrepareRetryInterval:    5,
			MaxCachedLength:         10,
			ProposalRetryInterval:   5,
		},
		NumCommands: requests,
	}

	c := types.NewComparison(rsl.CoverageAnalyzer(), rsl.CoverageComparator(saveFile))
	c.AddExperiment(types.NewExperiment(
		"random",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      types.NewRandomPolicy(),
			Environment: GetRSLEnvironment(config),
		},
	))
	strictPolicy := policies.NewStrictPolicy(types.NewRandomPolicy())
	strictPolicy.AddPolicy(policies.If(policies.Always()).Then(types.PickKeepSame()))

	c.AddExperiment(types.NewExperiment(
		"Strict",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      policies.NewStrictPolicy(strictPolicy),
			Environment: GetRSLEnvironment(config),
		},
	))
	c.AddExperiment(types.NewExperiment(
		"BonusMax",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      policies.NewBonusPolicyGreedy(0.1, 0.99, 0.2),
			Environment: GetRSLEnvironment(config),
		},
	))

	c.Run()
}

func GetRSLEnvironment(c rsl.RSLEnvConfig) types.Environment {
	colors := []rsl.RSLColorFunc{rsl.ColorState(), rsl.ColorDecree(), rsl.ColorDecided()}
	return types.NewPartitionEnv(types.PartitionEnvConfig{
		Painter:                rsl.NewRSLPainter(colors...),
		Env:                    rsl.NewRLSPartitionEnv(c),
		NumReplicas:            c.Nodes,
		TicketBetweenPartition: 3,
		MaxMessagesPerTick:     3,
	})
}

func RSLExplorationCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "rsl",
		Run: func(cmd *cobra.Command, args []string) {
			RSLExploration()
		},
	}
	cmd.PersistentFlags().IntVarP(&requests, "requests", "r", 1, "Number of requests to run with")
	return cmd
}
