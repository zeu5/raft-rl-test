package main

import (
	"strconv"

	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/grid"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/types"
)

func GridRewardMachine(episodes, horizon int, height, width, grids int) {

	allRewards := []types.RewardFunc{}
	rm := policies.NewRewardMachine()
	for i := 1; i < grids; i++ {
		pred := grid.ToGrid(i)
		rm.On(pred, "Grid"+strconv.Itoa(i))
		allRewards = append(allRewards, pred)
	}

	c := types.NewComparison(policies.RewardMachineAnalyzer(allRewards, grid.DefaultStateAbstractor()), policies.RewardMachineCoverageComparator())

	c.AddExperiment(types.NewExperiment(
		"Random",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      types.NewRandomPolicy(),
			Environment: grid.NewGridEnvironment(height, width, grids),
		},
	))
	c.AddExperiment(types.NewExperiment(
		"Biased-Policy",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      policies.NewGuidedPolicy(grid.NextGrid(height, width), 0.3, 0.95, 0.1),
			Environment: grid.NewGridEnvironment(height, width, grids),
		},
	))
	c.AddExperiment(types.NewExperiment(
		"Reward-Machine",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      policies.NewRewardMachinePolicy(rm),
			Environment: grid.NewGridEnvironment(height, width, grids),
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
