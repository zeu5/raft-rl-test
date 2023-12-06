package benchmarks

import (
	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/lpaxos"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/types"
)

func PaxosPart(episodes, horizon int, saveFile string, requests int) {
	lPaxosConfig := lpaxos.LPaxosEnvConfig{
		Replicas: 3,
		Requests: requests,
		Timeout:  12,
		Timeouts: timeouts,
	}

	c := types.NewComparison(runs, saveFile, false)
	c.AddAnalysis("Plot", lpaxos.LPaxosAnalyzer(saveFile), lpaxos.LPaxosComparator(saveFile))

	c.AddExperiment(types.NewExperiment(
		"Random-Part",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      types.NewRandomPolicy(),
			Environment: getLPaxosPartEnv(lPaxosConfig, true),
		},
	))
	c.AddExperiment(types.NewExperiment(
		"NegReward-Part",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      types.NewSoftMaxNegPolicy(0.1, 0.99, 1),
			Environment: getLPaxosPartEnv(lPaxosConfig, true),
		},
	))
	c.AddExperiment(types.NewExperiment(
		"BonusMaxRL-Part",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      policies.NewBonusPolicyGreedy(0.1, 0.99, 0.2),
			Environment: getLPaxosPartEnv(lPaxosConfig, true),
		},
	))

	c.Run()
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
			PaxosPart(episodes, horizon, saveFile, r)
		},
	}
	cmd.PersistentFlags().IntVarP(&r, "requests", "r", 1, "Number of requests to run with")
	cmd.PersistentFlags().BoolVarP(&timeouts, "timeouts", "t", false, "Run with timeouts or not")
	return cmd
}
