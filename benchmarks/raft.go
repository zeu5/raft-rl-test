package benchmarks

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/raft"
	"github.com/zeu5/raft-rl-test/types"
)

var timeouts bool
var requests int
var abstracter string
var hierarchy string

func Raft(episodes, horizon int, saveFile string, ctx context.Context) {
	raftConfig := raft.RaftEnvironmentConfig{
		Replicas:      5,
		ElectionTick:  15,
		HeartbeatTick: 3,
		Timeouts:      timeouts,
		Requests:      requests,
	}
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
	c.AddAnalysis("Plot", raft.RaftAnalyzerCtor(saveFile), raft.RaftPlotComparator(saveFile))
	c.AddExperiment(types.NewExperiment("RL", types.NewSoftMaxNegPolicy(0.3, 0.7, 1), getRaftEnv(raftConfig, abstracter)))
	c.AddExperiment(types.NewExperiment("Random", types.NewRandomPolicy(), getRaftEnv(raftConfig, abstracter)))
	c.AddExperiment(types.NewExperiment("BonusMaxRL", policies.NewBonusPolicyGreedy(0.1, 0.99, 0), getRaftEnv(raftConfig, abstracter)))

	c.Run(ctx)
}

func getRaftEnv(config raft.RaftEnvironmentConfig, abstractor string) types.Environment {
	if abstractor == "none" {
		return raft.NewRaftEnvironment(config)
	}
	var abs raft.StateAbstracter
	switch abstractor {
	case "ignore-vote":
		abs = raft.IgnoreVote()
	case "ignore-term":
		abs = raft.IgnoreTerm()
	case "ignore-term-nonleader":
		abs = raft.IgnoreTermUnlessLeader()
	default:
		abs = raft.DefaultAbstractor()
	}
	return raft.NewAbsRaftEnvironment(config, abs)
}

func RaftCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "raft",
		Run: func(cmd *cobra.Command, args []string) {
			Raft(episodes, horizon, saveFile, context.Background())
		},
	}
	cmd.PersistentFlags().StringVarP(&abstracter, "abstractor", "a", "none", "Abstraction to use")
	cmd.PersistentFlags().IntVarP(&requests, "requests", "r", 1, "Number of requests to run with")
	cmd.PersistentFlags().BoolVarP(&timeouts, "timeouts", "t", false, "Run with timeouts or not")
	return cmd
}
