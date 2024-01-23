package benchmarks

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/grid"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/types"
)

func GridReward(episodes, horizon int, saveFile string, height, width, grids int, runs int, ctx context.Context) {

	doors := []grid.Door{
		// from grid 0
		{From: grid.Position{I: 35, J: 35, K: 0}, To: grid.Position{I: 0, J: 0, K: 1}},
		// {From: grid.Position{I: 4, J: 4, K: 0}, To: grid.Position{I: 0, J: 0, K: 2}},
		// {From: grid.Position{I: 5, J: 5, K: 0}, To: grid.Position{I: 0, J: 0, K: 3}},
		// {From: grid.Position{I: 16, J: 22, K: 0}, To: grid.Position{I: 0, J: 0, K: 2}},
		// {From: grid.Position{I: 31, J: 23, K: 0}, To: grid.Position{I: 0, J: 0, K: 3}},
		// {From: grid.Position{I: 26, J: 5, K: 0}, To: grid.Position{I: 0, J: 0, K: 1}},
		// {From: grid.Position{I: 12, J: 32, K: 0}, To: grid.Position{I: 0, J: 0, K: 2}},
		// {From: grid.Position{I: 10, J: 10, K: 0}, To: grid.Position{I: 0, J: 0, K: 3}},
		// {From: grid.Position{I: 9, J: 6, K: 0}, To: grid.Position{I: 0, J: 0, K: 4}},

		// from grid 1
		{From: grid.Position{I: 35, J: 35, K: 1}, To: grid.Position{I: 0, J: 0, K: 2}},
		// {From: grid.Position{I: 1, J: 5, K: 1}, To: grid.Position{I: 0, J: 0, K: 3}},
		// {From: grid.Position{I: 16, J: 22, K: 1}, To: grid.Position{I: 0, J: 0, K: 2}},
		// {From: grid.Position{I: 31, J: 23, K: 1}, To: grid.Position{I: 0, J: 0, K: 3}},
		// {From: grid.Position{I: 12, J: 32, K: 1}, To: grid.Position{I: 0, J: 0, K: 2}},
		// {From: grid.Position{I: 10, J: 10, K: 1}, To: grid.Position{I: 0, J: 0, K: 3}},
		// {From: grid.Position{I: 9, J: 6, K: 1}, To: grid.Position{I: 0, J: 0, K: 4}},

		// from grid 2
		{From: grid.Position{I: 35, J: 35, K: 2}, To: grid.Position{I: 0, J: 0, K: 3}},
		// {From: grid.Position{I: 31, J: 23, K: 2}, To: grid.Position{I: 0, J: 0, K: 3}},
		// {From: grid.Position{I: 10, J: 10, K: 2}, To: grid.Position{I: 0, J: 0, K: 3}},
		// {From: grid.Position{I: 60, J: 60, K: 2}, To: grid.Position{I: 0, J: 0, K: 4}},

		// from grid 3
		{From: grid.Position{I: 35, J: 35, K: 3}, To: grid.Position{I: 0, J: 0, K: 4}},
	}

	c := types.NewComparison(&types.ComparisonConfig{
		Runs:       runs,
		Episodes:   episodes,
		Horizon:    horizon,
		RecordPath: saveFile,
		Timeout:    0 * time.Second,
		// record flags
		RecordTraces: false,
		RecordTimes:  false,
		RecordPolicy: false,
		// last traces
		PrintLastTraces:     0,
		PrintLastTracesFunc: nil,
		// report config
		ReportConfig: types.RepConfigOff(),
	})
	// c.AddAnalysis("GridPlot", grid.GridAnalyzer, grid.GridDepthComparator())
	c.AddAnalysis("Coverage", grid.NewGridCoverageAnalyzer(), grid.GridCoverageComparator())

	c.AddExperiment(types.NewExperiment(
		"Random-Part",
		types.NewRandomPolicy(),
		grid.NewGridEnvironment(height, width, grids, doors...),
	))
	c.AddExperiment(types.NewExperiment(
		"NegReward-Part",
		policies.NewSoftMaxNegFreqPolicy(0.3, 0.7, 1),
		grid.NewGridEnvironment(height, width, grids, doors...),
	))

	c.AddExperiment(types.NewExperiment(
		"Exploration-Policy",
		policies.NewBonusPolicyGreedy(0.1, 0.99, 0.02),
		grid.NewGridEnvironment(height, width, grids, doors...),
	))

	c.Run(ctx)
}

func GridRewardCommand() *cobra.Command {
	var height int
	var width int
	var grids int

	cmd := &cobra.Command{
		Use: "grid",
		Run: func(cmd *cobra.Command, args []string) {
			GridReward(episodes, horizon, saveFile, height, width, grids, runs, context.Background())
		},
	}
	cmd.PersistentFlags().IntVar(&height, "height", 40, "Height of each grid")
	cmd.PersistentFlags().IntVar(&width, "width", 40, "Width of each grid")
	cmd.PersistentFlags().IntVar(&grids, "grids", 5, "Number of grids")
	return cmd
}
