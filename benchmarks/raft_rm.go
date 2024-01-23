package benchmarks

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/raft"
	"github.com/zeu5/raft-rl-test/types"
)

func RaftRM(episodes, horizon int, savePath string, ctx context.Context) {
	// config of the running system
	raftConfig := raft.RaftEnvironmentConfig{
		Replicas:      3,
		ElectionTick:  10, // lower bound for a process to try to go to new term (starting an election) - double of this is upperbound
		HeartbeatTick: 3,  // frequency of heartbeats
		Timeouts:      timeouts,
		Requests:      requests,
	}

	// abstraction for both plot and RL
	// colors is one abstraction definition
	colors := []raft.RaftColorFunc{raft.ColorState(), raft.ColorCommit(), raft.ColorLeader(), raft.ColorVote(), raft.ColorBoundedTerm(5)}

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
	c.AddAnalysis("Plot", raft.NewRaftAnalyzer(saveFile, colors...), raft.RaftPlotComparator(saveFile))

	// here you add different policies with their parameters
	c.AddExperiment(types.NewExperiment("RL", policies.NewSoftMaxNegFreqPolicy(0.3, 0.7, 1), getRaftPartEnv(raftConfig, colors)))
	c.AddExperiment(types.NewExperiment("Random", types.NewRandomPolicy(), getRaftPartEnv(raftConfig, colors)))
	c.AddExperiment(types.NewExperiment("BonusMaxRL", policies.NewBonusPolicyGreedy(0.1, 0.99, 0.2), getRaftPartEnv(raftConfig, colors)))

	c.Run(ctx)
}

func RaftRMCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "raft-rm",
		Run: func(cmd *cobra.Command, args []string) {
			RaftRM(episodes, horizon, saveFile, context.Background())
		},
	}
	cmd.PersistentFlags().IntVarP(&requests, "requests", "r", 1, "Number of requests to run with")
	return cmd
}
