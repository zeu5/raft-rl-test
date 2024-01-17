package benchmarks

import (
	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/raft"
	"github.com/zeu5/raft-rl-test/types"
)

var timeouts bool
var requests int
var abstracter string
var hierarchy string

func Raft(episodes, horizon int, saveFile string) {
	raftConfig := raft.RaftEnvironmentConfig{
		Replicas:      5,
		ElectionTick:  15,
		HeartbeatTick: 3,
		Timeouts:      timeouts,
		Requests:      requests,
	}
	c := types.NewComparison(runs, saveFile, false)
	c.AddAnalysis("Plot", raft.RaftAnalyzer(saveFile), raft.RaftPlotComparator(saveFile))
	c.AddExperiment(types.NewExperiment("RL", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      types.NewSoftMaxNegPolicy(0.3, 0.7, 1),
		Environment: getRaftEnv(raftConfig, abstracter),
	}, types.RepConfigOff()))
	c.AddExperiment(types.NewExperiment("Random", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      types.NewRandomPolicy(),
		Environment: getRaftEnv(raftConfig, abstracter),
	}, types.RepConfigOff()))
	c.AddExperiment(types.NewExperiment("BonusMaxRL", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      policies.NewBonusPolicyGreedy(0.1, 0.99, 0),
		Environment: getRaftEnv(raftConfig, abstracter),
	}, types.RepConfigOff()))

	c.Run()
}

func getRaftEnv(config raft.RaftEnvironmentConfig, abstractor string) types.EnvironmentUnion {
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
			Raft(episodes, horizon, saveFile)
		},
	}
	cmd.PersistentFlags().StringVarP(&abstracter, "abstractor", "a", "none", "Abstraction to use")
	cmd.PersistentFlags().IntVarP(&requests, "requests", "r", 1, "Number of requests to run with")
	cmd.PersistentFlags().BoolVarP(&timeouts, "timeouts", "t", false, "Run with timeouts or not")
	return cmd
}
