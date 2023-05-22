package main

import (
	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/lpaxos"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/types"
)

func PaxosPart(episodes, horizon int, saveFile string) {
	lPaxosConfig := lpaxos.LPaxosEnvConfig{
		Replicas: 3,
		Requests: requests,
		Timeout:  12,
		Timeouts: timeouts,
	}
	c := types.NewComparison(lpaxos.LPaxosAnalyzer(saveFile), lpaxos.LPaxosComparator(saveFile))
	c.AddExperiment(types.NewExperiment(
		"RL",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      types.NewSoftMaxNegPolicy(0.3, 0.7, 1),
			Environment: getLPaxosPartEnv(lPaxosConfig),
		},
	))
	c.AddExperiment(types.NewExperiment(
		"Random",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      types.NewRandomPolicy(),
			Environment: getLPaxosPartEnv(lPaxosConfig),
		},
	))
	c.AddExperiment(types.NewExperiment(
		"BonusMaxRL",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      policies.NewBonusPolicyGreedy(horizon, 0.99, 0.2),
			Environment: getLPaxosPartEnv(lPaxosConfig),
		},
	))
	c.AddExperiment(types.NewExperiment(
		"BonusSoftMaxRL",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      policies.NewBonusPolicySoftMax(horizon, 0.99, 0.01),
			Environment: getLPaxosPartEnv(lPaxosConfig),
		},
	))

	c.Run()
}

func getLPaxosPartEnv(config lpaxos.LPaxosEnvConfig) types.Environment {
	return types.NewPartitionEnv(types.PartitionEnvConfig{
		Painter:                &lpaxos.LNodeStatePainter{},
		Env:                    lpaxos.NewLPaxosPartitionEnv(config),
		NumReplicas:            config.Replicas,
		TicketBetweenPartition: 5,
		MaxMessagesPerTick:     3,
	})
}

func PaxosPartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "paxos-part",
		Run: func(cmd *cobra.Command, args []string) {
			PaxosPart(episodes, horizon, saveFile)
		},
	}
	cmd.PersistentFlags().IntVarP(&requests, "requests", "r", 1, "Number of requests to run with")
	cmd.PersistentFlags().BoolVarP(&timeouts, "timeouts", "t", false, "Run with timeouts or not")
	return cmd
}
