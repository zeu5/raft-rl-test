//go:build exclude

package benchmarks

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/lpaxos"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/types"
)

func PaxosReward(episodes, horizon int, saveFile string, ctx context.Context) {
	lPaxosConfig := lpaxos.LPaxosEnvConfig{
		Replicas: 3,
		Requests: requests,
		Timeout:  12,
		Timeouts: timeouts,
	}

	commit := lpaxos.Commit()

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
	c.AddAnalysis("Reward", lpaxos.NewRewardStatesVisitedAnalyzer([]string{"commit"}, []types.RewardFunc{commit}, saveFile), lpaxos.RewardStateComparator())
	c.AddExperiment(types.NewExperiment(
		"Random-Part",
		types.NewRandomPolicy(),
		getLPaxosPartEnv(lPaxosConfig, true),
	))
	c.AddExperiment(types.NewExperiment(
		"Biased-Policy",
		policies.NewGuidedPolicy(nil, 0.2, 0.95, 0.02),
		getLPaxosPartEnv(lPaxosConfig, true),
	))

	strictPolicy := policies.NewStrictPolicy(types.NewRandomPolicy())
	strictPolicy.AddPolicy(policies.If(policies.Always()).Then(types.PickKeepSame()))

	c.AddExperiment(types.NewExperiment(
		"Strict-Policy",
		strictPolicy,
		getLPaxosPartEnv(lPaxosConfig, true),
	))

	c.Run(ctx)
}

func PaxosRewardCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "paxos-reward",
		Run: func(cmd *cobra.Command, args []string) {
			PaxosReward(episodes, horizon, saveFile, context.Background())
		},
	}
	cmd.PersistentFlags().IntVarP(&requests, "requests", "r", 1, "Number of requests to run with")
	cmd.PersistentFlags().BoolVarP(&timeouts, "timeouts", "t", false, "Run with timeouts or not")
	return cmd
}
