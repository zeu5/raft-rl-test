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

	// doors := []grid.Door{
	// 	{From: grid.Position{I: 10, J: 10, K: 0}, To: grid.Position{I: 0, J: 0, K: grids - 1}},
	// }

	// possible set of doors, designed for 40x40 grids, 5 grids
	doors := []grid.Door{
		// from grid 0
		{From: grid.Position{I: 2, J: 3, K: 0}, To: grid.Position{I: 0, J: 0, K: 1}},
		// {From: grid.Position{I: 4, J: 4, K: 0}, To: grid.Position{I: 0, J: 0, K: 2}},
		// {From: grid.Position{I: 5, J: 5, K: 0}, To: grid.Position{I: 0, J: 0, K: 3}},
		{From: grid.Position{I: 16, J: 22, K: 0}, To: grid.Position{I: 0, J: 0, K: 2}},
		{From: grid.Position{I: 31, J: 23, K: 0}, To: grid.Position{I: 0, J: 0, K: 3}},
		{From: grid.Position{I: 26, J: 5, K: 0}, To: grid.Position{I: 0, J: 0, K: 1}},
		{From: grid.Position{I: 12, J: 32, K: 0}, To: grid.Position{I: 0, J: 0, K: 2}},
		{From: grid.Position{I: 10, J: 10, K: 0}, To: grid.Position{I: 0, J: 0, K: 3}},
		// {From: grid.Position{I: 9, J: 6, K: 0}, To: grid.Position{I: 0, J: 0, K: 4}},

		// from grid 1
		{From: grid.Position{I: 5, J: 5, K: 1}, To: grid.Position{I: 0, J: 0, K: 2}},
		// {From: grid.Position{I: 1, J: 5, K: 1}, To: grid.Position{I: 0, J: 0, K: 3}},
		{From: grid.Position{I: 16, J: 22, K: 1}, To: grid.Position{I: 0, J: 0, K: 2}},
		{From: grid.Position{I: 31, J: 23, K: 1}, To: grid.Position{I: 0, J: 0, K: 3}},
		{From: grid.Position{I: 12, J: 32, K: 1}, To: grid.Position{I: 0, J: 0, K: 2}},
		{From: grid.Position{I: 10, J: 10, K: 1}, To: grid.Position{I: 0, J: 0, K: 3}},
		// {From: grid.Position{I: 9, J: 6, K: 1}, To: grid.Position{I: 0, J: 0, K: 4}},

		// from grid 2
		{From: grid.Position{I: 1, J: 5, K: 2}, To: grid.Position{I: 0, J: 0, K: 3}},
		{From: grid.Position{I: 31, J: 23, K: 2}, To: grid.Position{I: 0, J: 0, K: 3}},
		{From: grid.Position{I: 10, J: 10, K: 2}, To: grid.Position{I: 0, J: 0, K: 3}},
		{From: grid.Position{I: 60, J: 60, K: 2}, To: grid.Position{I: 0, J: 0, K: 4}},

		// from grid 3
		{From: grid.Position{I: 60, J: 60, K: 3}, To: grid.Position{I: 0, J: 0, K: 4}},
	}

	rm := policies.NewRewardMachine(grid.ReachGrid(4))
	// rm.On(grid.TakesDoor(doors[0]), policies.FinalState)
	// rm.On(grid.ReachGrid(4), policies.FinalState)
	rm.AddState(grid.ReachGrid(1), "Grid1")
	// rm.AddState(grid.GridAndPos_1_2(), "Grid01_2")
	// rm.AddState(grid.GridAndPos_1_4(), "Grid01_4")
	rm.AddState(grid.ReachGrid(2), "Grid2")
	// rm.AddState(grid.GridAndPos_23_20(), "Grid23_20")
	// rm.AddState(grid.GridAndPos_23_30(), "Grid23_30")
	// rm.AddState(grid.GridAndPos_23_40(), "Grid23_40")
	rm.AddState(grid.GridAndPos_23_50(), "Grid23_50")

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
			Policy:      policies.NewBonusPolicyGreedy(0.1, 0.99, 0.05),
			Environment: grid.NewGridEnvironment(height, width, grids, doors...),
		},
	))
	// c.AddExperiment(types.NewExperiment(
	// 	"Exploration-Softmax",
	// 	&types.AgentConfig{
	// 		Episodes:    episodes,
	// 		Horizon:     horizon,
	// 		Policy:      policies.NewBonusPolicySoftMax(0.1, 0.99, 10000.0),
	// 		Environment: grid.NewGridEnvironment(height, width, grids, doors...),
	// 	},
	// ))
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
