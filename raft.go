package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/raft"
	"github.com/zeu5/raft-rl-test/types"
)

var timeouts bool
var requests int
var abstracter string

func Raft(episodes, horizon int, saveFile string) {
	fmt.Printf("Running with timeout value: %v, requests: %d, abstraction: %s\n", timeouts, requests, abstracter)
	raftConfig := raft.RaftEnvironmentConfig{
		Replicas:      5,
		ElectionTick:  15,
		HeartbeatTick: 3,
		Timeouts:      timeouts,
		TicksPerStep:  1,
		Requests:      requests,
	}
	c := types.NewComparison(raft.RaftAnalyzer(saveFile), raft.RaftPlotComparator(saveFile, raft.ChainFilters(raft.MinCutOff(100), raft.Log())))
	c.AddExperiment(types.NewExperiment("RL", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      types.NewSoftMaxNegPolicy(0.3, 0.7, 1),
		Environment: getRaftEnv(raftConfig, abstracter),
	}))
	c.AddExperiment(types.NewExperiment("Random", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      types.NewRandomPolicy(),
		Environment: getRaftEnv(raftConfig, abstracter),
	}))
	c.AddExperiment(types.NewExperiment("BonusMaxRL", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      policies.NewBonusPolicyGreedy(0.1, 0.99, 0),
		Environment: getRaftEnv(raftConfig, abstracter),
	}))

	c.Run()
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
			Raft(episodes, horizon, saveFile)
		},
	}
	cmd.PersistentFlags().StringVarP(&abstracter, "abstractor", "a", "none", "Abstraction to use")
	cmd.PersistentFlags().IntVarP(&requests, "requests", "r", 1, "Number of requests to run with")
	cmd.PersistentFlags().BoolVarP(&timeouts, "timeouts", "t", false, "Run with timeouts or not")
	return cmd
}
