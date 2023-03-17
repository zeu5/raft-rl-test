package main

import (
	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/raft"
	"github.com/zeu5/raft-rl-test/rl"
)

func Two(episodes, horizon int, savefile string) {
	raftConfig := raft.RaftEnvironmentConfig{
		Replicas:      3,
		ElectionTick:  10,
		HeartbeatTick: 1,
		Timeouts:      true,
	}

	leaderElectedProperty := raft.LeaderElected()
	c := rl.NewComparison(raft.RaftAnalyzer, raft.RaftPlotComparator(saveFile))
	c.AddExperiment(rl.NewExperimentWithProperties(
		"RL",
		&rl.AgentConfig{
			Episodes: episodes,
			Horizon:  horizon,
			Policy: policies.NewPropertyGuidedPolicy(
				[]*rl.Monitor{leaderElectedProperty},
				0.3,
				0.7,
				0.2,
			),
			Environment: raft.NewRaftEnvironment(raftConfig),
		},
		[]*rl.Monitor{leaderElectedProperty},
	))
	c.AddExperiment(rl.NewExperimentWithProperties(
		"Random",
		&rl.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      rl.NewRandomPolicy(),
			Environment: raft.NewRaftEnvironment(raftConfig),
		},
		[]*rl.Monitor{leaderElectedProperty},
	))

	c.Run()
}

func TwoCommand() *cobra.Command {
	return &cobra.Command{
		Use: "two",
		Run: func(cmd *cobra.Command, args []string) {
			Two(episodes, horizon, saveFile)
		},
	}
}
