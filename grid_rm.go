package main

import (
	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/grid"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/types"
)

func GridRewardMachine(episodes, horizon int, height, width, grids int) {

	// rm := policies.NewRewardMachine()
	// for i := 1; i < grids-1; i++ {
	// 	pred := grid.ToGrid(i)
	// 	rm.On(pred, "Grid"+strconv.Itoa(i))
	// }
	// rm.On(grid.ToGrid(grids-1), policies.FinalState)

	doors := []grid.Door{
		{From: grid.Position{I: height / 2, J: width / 2, K: 0}, To: grid.Position{I: 0, J: 0, K: grids - 1}},
	}

	rm := policies.NewRewardMachine(horizon)
	rm.On(grid.ToGrid(grids-1), policies.FinalState)

	c := types.NewComparison(grid.GridAnalyzer, grid.GridDepthComparator())

	c.AddExperiment(types.NewExperiment(
		"Random",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      types.NewRandomPolicy(),
			Environment: grid.NewGridEnvironment(height, width, grids, doors...),
		},
	))
	c.AddExperiment(types.NewExperiment(
		"Exploration",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      policies.NewBonusPolicyGreedy(horizon, 0.99, 0.2),
			Environment: grid.NewGridEnvironment(height, width, grids, doors...),
		},
	))
	c.AddExperiment(types.NewExperiment(
		"Reward-Machine",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      policies.NewRewardMachinePolicy(rm),
			Environment: grid.NewGridEnvironment(height, width, grids, doors...),
		},
	))

	c.Run()
}

func GridRewardMachineCommand() *cobra.Command {
	var height int
	var width int
	var grids int

	cmd := &cobra.Command{
		Use: "grid-reward-rm",
		Run: func(cmd *cobra.Command, args []string) {
			GridRewardMachine(episodes, horizon, height, width, grids)
		},
	}
	cmd.PersistentFlags().IntVar(&height, "height", 5, "Height of each grid")
	cmd.PersistentFlags().IntVar(&width, "width", 5, "Width of each grid")
	cmd.PersistentFlags().IntVar(&grids, "grids", 2, "Number of grids")
	return cmd
}
