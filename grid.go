package main

import (
	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/grid"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/types"
)

func GridReward(episodes, horizon int, saveFile string) {

	inPosition := grid.InPosition(3, 4)

	c := types.NewComparison(grid.GridAnalyzer, grid.GridPositionComparator(3, 4))

	c.AddExperiment(types.NewExperiment(
		"Random-Part",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      types.NewRandomPolicy(),
			Environment: grid.NewGridEnvironment(5, 5),
		},
	))
	c.AddExperiment(types.NewExperiment(
		"Biased-Policy",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      policies.NewGuidedPolicy([]types.RewardFunc{inPosition}, 0.3, 0.95, 0.1),
			Environment: grid.NewGridEnvironment(5, 5),
		},
	))

	c.Run()
}

func GridRewardCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "grid-reward",
		Run: func(cmd *cobra.Command, args []string) {
			GridReward(episodes, horizon, saveFile)
		},
	}
	return cmd
}
