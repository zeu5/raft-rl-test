package main

import (
	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/grid"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/types"
)

func GridReward(episodes, horizon int, saveFile string, height, width, grids int) {

	c := types.NewComparison(grid.GridAnalyzer, grid.GridDepthComparator())

	c.AddExperiment(types.NewExperiment(
		"Random-Part",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      types.NewRandomPolicy(),
			Environment: grid.NewGridEnvironment(height, width, grids),
		},
	))
	c.AddExperiment(types.NewExperiment(
		"Exploration-Policy",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      policies.NewBonusPolicyGreedy(horizon, 0.99, 0.02),
			Environment: grid.NewGridEnvironment(height, width, grids),
		},
	))

	c.Run()
}

func GridRewardCommand() *cobra.Command {
	var height int
	var width int
	var grids int

	cmd := &cobra.Command{
		Use: "grid",
		Run: func(cmd *cobra.Command, args []string) {
			GridReward(episodes, horizon, saveFile, height, width, grids)
		},
	}
	cmd.PersistentFlags().IntVar(&height, "height", 5, "Height of each grid")
	cmd.PersistentFlags().IntVar(&width, "width", 5, "Width of each grid")
	cmd.PersistentFlags().IntVar(&grids, "grids", 2, "Number of grids")
	return cmd
}
