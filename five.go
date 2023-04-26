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
		TicksPerStep:  2,
	}

	leaderElectedProperty := raft.LeaderElected()
	// leaderCommittedProperty := raft.LeaderCommittedRequest()
	c := types.NewComparison(raft.RaftAnalyzer(savefile), raft.RaftPlotComparator(saveFile, raft.DefaultFilter()))
	c.AddExperiment(types.NewExperimentWithProperties(
		"RLManyPolicies",
		&types.AgentConfig{
			Episodes: episodes,
			Horizon:  horizon,
			Policy: policies.NewPropertyGuidedPolicy(
				[]*types.Monitor{leaderElectedProperty},
				0.3,
				0.7,
				0,
			),
			Environment: raft.NewRaftEnvironment(raftConfig),
		},
		[]*types.Monitor{leaderElectedProperty},
	))
	c.AddExperiment(types.NewExperimentWithProperties(
		"RLSinglePolicy",
		&types.AgentConfig{
			Episodes: episodes,
			Horizon:  horizon,
			Policy: policies.NewGuidedPolicy(
				leaderElectedProperty,
				0.3,
				0.7,
				0,
			),
			Environment: raft.NewRaftEnvironment(raftConfig),
		},
		[]*types.Monitor{leaderElectedProperty},
	))
	c.AddExperiment(types.NewExperimentWithProperties(
		"Random",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      types.NewRandomPolicy(),
			Environment: raft.NewRaftEnvironment(raftConfig),
		},
		[]*types.Monitor{leaderElectedProperty},
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
