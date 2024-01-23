package benchmarks

import (
	"context"
	"path"
	"time"

	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/rsl"
	"github.com/zeu5/raft-rl-test/types"
)

// Run exploration of the RSL algorithm
func RSLExploration(ctx context.Context) {
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

	colors := []rsl.RSLColorFunc{rsl.ColorState(), rsl.ColorDecree(), rsl.ColorDecided(), rsl.ColorBoundedBallot(5), rsl.ColorLogLength()}

	c := types.NewComparison(&types.ComparisonConfig{
		Runs:       runs,
		Episodes:   episodes,
		Horizon:    horizon,
		RecordPath: saveFile,
		Timeout:    0 * time.Second,
		// record flags
		RecordTraces: false,
		RecordTimes:  false,
		RecordPolicy: false,
		// last traces
		PrintLastTraces:     0,
		PrintLastTracesFunc: nil,
		// report config
		ReportConfig: types.RepConfigOff(),
	})
	c.AddAnalysis("Plot", rsl.NewCoverageAnalyzer(colors...), rsl.CoverageComparator(saveFile))
	c.AddAnalysis("Bugs", types.NewBugAnalyzer(
		path.Join(saveFile, "bugs"),
		types.BugDesc{Name: "InconsistentLogs", Check: rsl.InconsistentLogs()},
		types.BugDesc{Name: "MultiplePrimaries", Check: rsl.MultiplePrimaries()},
	), types.BugComparator(saveFile))
	c.AddAnalysis("Crashes", types.NewCrashAnalyzer(), types.CrashComparator(saveFile))
	c.AddAnalysis("PureCoverage", types.NewPureCoverageAnalyzer(), types.PureCoveragePlotter(saveFile))
	// Random exploration
	c.AddExperiment(types.NewExperiment(
		"Random",
		types.NewRandomPolicy(),
		GetRSLEnvironment(config, colors),
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
	// 		Environment: GetRSLEnvironment(config, colors),
	// 	},
	// ))
	// RL based exploration
	c.AddExperiment(types.NewExperiment(
		"NegReward",
		policies.NewSoftMaxNegFreqPolicy(0.1, 0.99, 1),
		GetRSLEnvironment(config, colors),
	))
	c.AddExperiment(types.NewExperiment(
		"BonusMax",
		policies.NewBonusPolicyGreedy(0.1, 0.99, 0.05),
		GetRSLEnvironment(config, colors),
	))

	c.Run(ctx)
}

func GetRSLEnvironment(c rsl.RSLEnvConfig, colors []rsl.RSLColorFunc) types.Environment {
	return types.NewPartitionEnv(types.PartitionEnvConfig{
		Painter:                rsl.NewRSLPainter(colors...),
		Env:                    rsl.NewRLSPartitionEnv(c),
		NumReplicas:            c.Nodes,
		TicketBetweenPartition: 3,
		MaxMessagesPerTick:     100,
		StaySameStateUpto:      5,
		WithCrashes:            true,
		CrashLimit:             100,
	})
}

func RSLExplorationCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "rsl",
		Run: func(cmd *cobra.Command, args []string) {
			RSLExploration(context.Background())
		},
	}
	cmd.PersistentFlags().IntVarP(&requests, "requests", "r", 3, "Number of requests to run with")
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
