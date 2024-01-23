package benchmarks

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/lpaxos"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/types"
)

func PaxosPart(episodes, horizon int, saveFile string, requests int, ctx context.Context) {
	lPaxosConfig := lpaxos.LPaxosEnvConfig{
		Replicas: 3,
		Requests: requests,
		Timeout:  12,
		Timeouts: timeouts,
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
	c.AddAnalysis("Plot", lpaxos.NewLPaxosAnalyzer(saveFile), lpaxos.LPaxosComparator(saveFile))

	c.AddExperiment(types.NewExperiment(
		"Random-Part",
		types.NewRandomPolicy(),
		getLPaxosPartEnv(lPaxosConfig, true),
	))
	c.AddExperiment(types.NewExperiment(
		"NegReward-Part",
		types.NewSoftMaxNegPolicy(0.1, 0.99, 1),
		getLPaxosPartEnv(lPaxosConfig, true),
	))
	c.AddExperiment(types.NewExperiment(
		"BonusMaxRL-Part",
		policies.NewBonusPolicyGreedy(0.1, 0.99, 0.2),
		getLPaxosPartEnv(lPaxosConfig, true),
	))

	c.Run(ctx)
}

func getLPaxosPartEnv(config lpaxos.LPaxosEnvConfig, part bool) types.Environment {
	if part {
		colors := []lpaxos.LColorFunc{lpaxos.ColorStep(), lpaxos.ColorDecided(), lpaxos.ColorLeader()}
		return types.NewPartitionEnv(types.PartitionEnvConfig{
			Painter:                lpaxos.NewLNodeStatePainter(colors...),
			Env:                    lpaxos.NewLPaxosPartitionEnv(config),
			NumReplicas:            config.Replicas,
			TicketBetweenPartition: 3,
			MaxMessagesPerTick:     3,
			StaySameStateUpto:      3,
		})
	}
	return lpaxos.NewLPaxosEnv(config)
}

func PaxosPartCommand() *cobra.Command {
	var r int
	cmd := &cobra.Command{
		Use:  "paxos",
		Long: "Run paxos pure exploration with partitions as environment",
		Run: func(cmd *cobra.Command, args []string) {
			PaxosPart(episodes, horizon, saveFile, r, context.Background())
		},
	}
	cmd.PersistentFlags().IntVarP(&r, "requests", "r", 1, "Number of requests to run with")
	cmd.PersistentFlags().BoolVarP(&timeouts, "timeouts", "t", false, "Run with timeouts or not")
	return cmd
}
