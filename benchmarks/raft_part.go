package benchmarks

// experiment file

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/raft"
	"github.com/zeu5/raft-rl-test/types"
)

func RaftPart(episodes, horizon int, saveFile string, ctx context.Context) {
	// config of the running system
	raftConfig := raft.RaftEnvironmentConfig{
		Replicas:          3,
		ElectionTick:      10, // lower bound for a process to try to go to new term (starting an election) - double of this is upperbound
		HeartbeatTick:     3,  // frequency of heartbeats
		Timeouts:          timeouts,
		Requests:          requests,
		SnapshotFrequency: 0,
	}

	// abstraction for both plot and RL
	// colors is one abstraction definition
	colors := []raft.RaftColorFunc{raft.ColorState(), raft.ColorCommit(), raft.ColorLeader(), raft.ColorVote(), raft.ColorBoundedTerm(10), raft.ColorLogLength()}

	// c is general experiment
	// colors ... , expanded list, can omit the argument
	// Analyzer takes the path to save data and colors... is the abstraction used to plot => makes the datasets
	// PlotComparator => makes plots from data
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

	// here you add different traces analysis and comparators -- to process traces into a dataset (analyzer) and output the results (comparator)
	c.AddAnalysis("Plot", raft.RaftAnalyzerCtor(saveFile, colors...), raft.RaftPlotComparator(saveFile))
	c.AddAnalysis("Crashes", types.CrashAnalyzerCtor(), types.CrashComparator(saveFile))
	c.AddAnalysis("PureCoverage", types.PureCoverageAnalyzerCtor(), types.PureCoveragePlotter(saveFile))
	// c.AddAnalysis("PartitionCoverage", types.PartitionCoverage(), types.PartitionCoveragePlotter(saveFile))

	// here you add different policies with their parameters
	// strict := policies.NewStrictPolicy(types.NewRandomPolicy())
	// strict.AddPolicy(policies.If(policies.Always()).Then(types.PickKeepSame()))

	// c.AddExperiment(types.NewExperiment("Strict", &types.AgentConfig{
	// 	Episodes:    episodes,
	// 	Horizon:     horizon,
	// 	Policy:      strict,
	// 	Environment: getRaftPartEnv(raftConfig, colors),
	// }))

	c.AddExperiment(types.NewExperiment("BonusMaxRL", policies.NewBonusPolicyGreedyReward(0.1, 0.99, 0.05), getRaftPartEnv(raftConfig, colors)))
	c.AddExperiment(types.NewExperiment("RL", policies.NewSoftMaxNegFreqPolicy(0.3, 0.7, 1), getRaftPartEnv(raftConfig, colors)))
	c.AddExperiment(types.NewExperiment("Random", types.NewRandomPolicy(), getRaftPartEnv(raftConfig, colors)))

	c.Run(ctx)
}

func getRaftPartEnv(config raft.RaftEnvironmentConfig, colors []raft.RaftColorFunc) types.Environment {
	return types.NewPartitionEnv(types.PartitionEnvConfig{
		Painter:                raft.NewRaftStatePainter(colors...),  // pass the abstraction to env
		Env:                    raft.NewPartitionEnvironment(config), // actual environment
		TicketBetweenPartition: 3,                                    // ticks between actions
		MaxMessagesPerTick:     100,                                  // upper bound of random num of delivered messages
		StaySameStateUpto:      5,                                    // counter to distinguish consecutive states
		NumReplicas:            config.Replicas,
		WithCrashes:            true,
		CrashLimit:             100,
	})
}

func getRaftPartEnvCfg(config raft.RaftEnvironmentConfig, colors []raft.RaftColorFunc, rlConfig raft.RLConfig) types.Environment {

	return types.NewPartitionEnv(types.PartitionEnvConfig{
		Painter:                raft.NewRaftStatePainter(colors...),  // pass the abstraction to env
		Env:                    raft.NewPartitionEnvironment(config), // actual environment
		TicketBetweenPartition: rlConfig.TicksBetweenPartition,       // ticks between actions
		MaxMessagesPerTick:     rlConfig.MaxMessagesPerTick,          // upper bound of random num of delivered messages
		StaySameStateUpto:      rlConfig.StaySameStateUpTo,           // counter to distinguish consecutive states
		NumReplicas:            config.Replicas,                      // number of replicas
		WithCrashes:            rlConfig.WithCrashes,                 // enable crash actions
		CrashLimit:             rlConfig.CrashLimit,                  // limit the number of crash actions
	})
}

func RaftPartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "raft-part",
		Run: func(cmd *cobra.Command, args []string) {
			RaftPart(episodes, horizon, saveFile, context.Background())
		},
	}
	cmd.PersistentFlags().IntVarP(&requests, "requests", "r", 3, "Number of requests to run with")
	return cmd
}
