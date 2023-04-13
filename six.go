package main

import (
	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/grid"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/types"
)

// Experiment with a 2D grid and providing rewards to check if the policy is working
func Six(episodes, horizon int, saveFile string) {
	height := 30
	width := 30

	property := grid.PosReached(10, 10)
	c := types.NewComparison(grid.GridAnalyzer, grid.GridPlotComparator(saveFile))
	c.AddExperiment(types.NewExperimentWithProperties(
		"RLManyProperties",
		&types.AgentConfig{
			Episodes: episodes,
			Horizon:  horizon,
			Policy: policies.NewPropertyGuidedPolicy(
				[]*types.Monitor{property},
				0.3,
				0.7,
				0,
			),
			Environment: grid.NewGridEnvironment(height, width),
		},
		[]*types.Monitor{property},
	))
	c.AddExperiment(types.NewExperimentWithProperties(
		"RLSinglePolicy",
		&types.AgentConfig{
			Episodes: episodes,
			Horizon:  horizon,
			Policy: policies.NewGuidedPolicy(
				property,
				0.3,
				0.7,
				0,
			),
			Environment: grid.NewGridEnvironment(height, width),
		},
		[]*types.Monitor{property},
	))
	c.AddExperiment(types.NewExperimentWithProperties(
		"Random",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      types.NewRandomPolicy(),
			Environment: grid.NewGridEnvironment(height, width),
		},
		[]*types.Monitor{property},
	))
	c.AddExperiment(types.NewExperimentWithProperties(
		"Neg",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      types.NewSoftMaxNegPolicy(0.3, 0.7),
			Environment: grid.NewGridEnvironment(height, width),
		},
		[]*types.Monitor{property},
	))

	c.Run()
}

func SixCommand() *cobra.Command {
	return &cobra.Command{
		Use: "six",
		Run: func(cmd *cobra.Command, args []string) {
			Six(episodes, horizon, saveFile)
		},
	}
}
