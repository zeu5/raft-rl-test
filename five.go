package main

import (
	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/raft"
	"github.com/zeu5/raft-rl-test/types"
)

func Five(episodes, horizon int, savefile string) {
	raftConfig := raft.RaftEnvironmentConfig{
		Replicas:      3,
		ElectionTick:  10,
		HeartbeatTick: 1,
		Timeouts:      true,
	}

	// leaderElectedProperty := raft.LeaderElected()
	leaderCommittedProperty := raft.LeaderCommittedRequest()
	c := types.NewComparison(raft.RaftAnalyzer, raft.RaftPlotComparator(saveFile))
	c.AddExperiment(types.NewExperimentWithProperties(
		"RL",
		&types.AgentConfig{
			Episodes: episodes,
			Horizon:  horizon,
			Policy: policies.NewGuidedPolicy(
				leaderCommittedProperty,
				0.3,
				0.7,
			),
			Environment: raft.NewRaftEnvironment(raftConfig),
		},
		[]*types.Monitor{leaderCommittedProperty},
	))
	c.AddExperiment(types.NewExperimentWithProperties(
		"Random",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      types.NewRandomPolicy(),
			Environment: raft.NewRaftEnvironment(raftConfig),
		},
		[]*types.Monitor{leaderCommittedProperty},
	))

	c.Run()
}

func FiveCommand() *cobra.Command {
	return &cobra.Command{
		Use: "five",
		Run: func(cmd *cobra.Command, args []string) {
			Five(episodes, horizon, saveFile)
		},
	}
}
