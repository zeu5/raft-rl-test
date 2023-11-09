package main

import (
	"path"

	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/rsl"
	"github.com/zeu5/raft-rl-test/types"
)

// Run exploration of the RSL algorithm
func RSLExploration() {
	// Config for the RSL environment
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
		NumCommands:        requests,
		AdditionalCommands: make([]rsl.Command, 0),
	}

	colors := []rsl.RSLColorFunc{rsl.ColorState(), rsl.ColorDecree(), rsl.ColorDecided(), rsl.ColorBoundedBallot(5)}

	c := types.NewComparison(runs)
	c.AddAnalysis("Plot", rsl.CoverageAnalyzer(colors...), rsl.CoverageComparator(saveFile))
	c.AddAnalysis("Bugs", types.BugAnalyzer(
		path.Join(saveFile, "bugs"),
		types.BugDesc{Name: "InconsistentLogs", Check: rsl.InconsistentLogs()},
		types.BugDesc{Name: "MultiplePrimaries", Check: rsl.MultiplePrimaries()},
	), types.BugComparator(saveFile))
	// Random exploration
	c.AddExperiment(types.NewExperiment(
		"Random",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      types.NewRandomPolicy(),
			Environment: GetRSLEnvironment(config, colors),
		},
	))
	// strictPolicy := policies.NewStrictPolicy(types.NewRandomPolicy())
	// strictPolicy.AddPolicy(policies.If(policies.Always()).Then(types.PickKeepSame()))

	// // Policy to never partition
	// c.AddExperiment(types.NewExperiment(
	// 	"Strict",
	// 	&types.AgentConfig{
	// 		Episodes:    episodes,
	// 		Horizon:     horizon,
	// 		Policy:      policies.NewStrictPolicy(strictPolicy),
	// 		Environment: GetRSLEnvironment(config),
	// 	},
	// ))
	// RL based exploration
	c.AddExperiment(types.NewExperiment(
		"NegReward",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      policies.NewSoftMaxNegFreqPolicy(0.1, 0.99, 1),
			Environment: GetRSLEnvironment(config, colors),
		},
	))
	c.AddExperiment(types.NewExperiment(
		"BonusMax",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      policies.NewBonusPolicyGreedy(0.1, 0.99, 0.2),
			Environment: GetRSLEnvironment(config, colors),
		},
	))

	c.Run()
}

func GetRSLEnvironment(c rsl.RSLEnvConfig, colors []rsl.RSLColorFunc) types.Environment {
	return types.NewPartitionEnv(types.PartitionEnvConfig{
		Painter:                rsl.NewRSLPainter(colors...),
		Env:                    rsl.NewRLSPartitionEnv(c),
		NumReplicas:            c.Nodes,
		TicketBetweenPartition: 3,
		MaxMessagesPerTick:     3,
		StaySameStateUpto:      5,
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

// Experiments to run
// 1. Find the tradeoff between ticks and stay same upto
// 2. Define reward machines with more than one state.
// 3. Check if variability is important for finding bugs.
// 4. Check if granularity matters to find a bug.

// Some scenarios
// Select a node to be primary (in a particular phase) and transition to a different primary (in a different phase)
// Select a node to be primary and wait for x commits with that node as primary
