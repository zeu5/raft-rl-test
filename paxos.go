package main

import (
	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/lpaxos"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/types"
)

func Paxos(episodes, horizon int, saveFile string) {
	// The configuration for the paxos environment
	lPaxosConfig := lpaxos.LPaxosEnvConfig{
		// Number of replicas to run
		Replicas: 3,
		// Number of initial requests to inject into the system
		Requests: requests,
		// The timeout value in terms of number of ticks
		Timeout: 12,
		// Timeouts is a boolean flag to indicate if the environment has drop message actions or not
		Timeouts: timeouts,
	}

	property := lpaxos.InconsistentLogs()
	// Comparison runs different agents as specified below. Then analyzes the traces for each agent configuration and compares them
	c := types.NewComparison(lpaxos.LPaxosAnalyzer(saveFile), lpaxos.LPaxosComparator(saveFile))
	// Adding the different policy and experiments
	c.AddExperiment(types.NewExperimentWithProperties(
		"RL",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      types.NewSoftMaxNegPolicy(0.3, 0.7, 1),
			Environment: getLPaxosEnv(lPaxosConfig, abstracter),
		},
		[]*types.Monitor{property},
	))
	c.AddExperiment(types.NewExperimentWithProperties(
		"Random",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      types.NewRandomPolicy(),
			Environment: getLPaxosEnv(lPaxosConfig, abstracter),
		},
		[]*types.Monitor{property},
	))
	c.AddExperiment(types.NewExperimentWithProperties(
		"BonusMaxRL",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      policies.NewBonusPolicyGreedy(0.1, 0.99, 0.2),
			Environment: getLPaxosEnv(lPaxosConfig, abstracter),
		},
		[]*types.Monitor{property},
	))
	c.AddExperiment(types.NewExperimentWithProperties(
		"BonusSoftMaxRL",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      policies.NewBonusPolicySoftMax(0.1, 0.99, 0.01),
			Environment: getLPaxosEnv(lPaxosConfig, abstracter),
		},
		[]*types.Monitor{property}))

	// Invoking the different experiments
	c.Run()
}

func getLPaxosEnv(config lpaxos.LPaxosEnvConfig, abs string) types.Environment {
	if abs == "none" {
		return lpaxos.NewLPaxosEnv(config)
	}
	abstracter := lpaxos.DefaultAbstractor()
	switch abs {
	case "ignore-phase":
		abstracter = lpaxos.IgnorePhase()
	case "ignore-last":
		abstracter = lpaxos.IgnoreLast()
	}
	return lpaxos.NewLPaxosAbsEnv(config, abstracter)
}

// PaxosCommand is the subcommand to run coverage benchmarks for the paxos example
// Refer to "Paxos" function for the exact algorithms to add
func PaxosCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "paxos",
		Run: func(cmd *cobra.Command, args []string) {
			Paxos(episodes, horizon, saveFile)
		},
	}
	cmd.PersistentFlags().StringVarP(&abstracter, "abstractor", "a", "", "Abstraction to use")
	cmd.PersistentFlags().IntVarP(&requests, "requests", "r", 1, "Number of requests to run with")
	cmd.PersistentFlags().BoolVarP(&timeouts, "timeouts", "t", false, "Run with timeouts or not")
	return cmd
}
