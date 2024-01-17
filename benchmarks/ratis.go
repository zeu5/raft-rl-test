package benchmarks

import (
	"context"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/ratis"
	"github.com/zeu5/raft-rl-test/types"
)

func RatisExploration(episodes, horizon int, saveFile string, ctx context.Context) {
	env := ratis.NewRatisRaftEnv(ctx, &ratis.RatisClusterConfig{
		NumNodes:            3,
		BasePort:            5000,
		BaseInterceptPort:   2023,
		InterceptListenPort: 7074,
		RatisJarPath:        "/Users/srinidhin/Local/github/berkay/ratis-fuzzing/ratis-examples/target/ratis-examples-2.5.1.jar",
		WorkingDir:          "/Users/srinidhin/Local/github/berkay/ratis-fuzzing",
		GroupID:             "02511d47-d67c-49a3-9011-abb3109a44c1",
	})
	colors := []ratis.RatisColorFunc{ratis.ColorState(), ratis.ColorCommit(), ratis.ColorLeader(), ratis.ColorVote(), ratis.ColorBoundedTerm(5)}

	partitionEnv := types.NewPartitionEnv(types.PartitionEnvConfig{
		Painter:                ratis.NewRatisStatePainter(colors...),
		Env:                    env,
		TicketBetweenPartition: 3,
		MaxMessagesPerTick:     3,
		StaySameStateUpto:      2,
		NumReplicas:            3,
		WithCrashes:            false,
	})

	c := types.NewComparison(runs, saveFile, false)

	c.AddAnalysis("plot", ratis.CoverageAnalyzer(colors...), ratis.CoverageComparator(saveFile))
	c.AddAnalysis("bugs", ratis.BugAnalyzer(saveFile), ratis.BugComparator())
	c.AddAnalysis("logs", ratis.LogAnalyzer(saveFile), types.NoopComparator())

	// c.AddExperiment(types.NewExperiment("NegReward", &types.AgentConfig{
	// 	Episodes:    episodes,
	// 	Horizon:     horizon,
	// 	Policy:      policies.NewSoftMaxNegFreqPolicy(0.1, 0.99, 1),
	// 	Environment: partitionEnv,
	// }))

	c.AddExperiment(types.NewExperiment("Random", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      types.NewRandomPolicy(),
		Environment: partitionEnv,
	}, types.RepConfigOff()))

	// c.AddExperiment(types.NewExperiment("BonusMax", &types.AgentConfig{
	// 	Episodes:    episodes,
	// 	Horizon:     horizon,
	// 	Policy:      policies.NewBonusPolicyGreedy(0.1, 0.99, 0.2),
	// 	Environment: partitionEnv,
	// }))

	c.Run()
	env.Cleanup()
}

func RatisExplorationCommand() *cobra.Command {
	return &cobra.Command{
		Use: "ratis",
		Run: func(cmd *cobra.Command, args []string) {
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt)

			doneCh := make(chan struct{})

			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				select {
				case <-sigCh:
				case <-doneCh:
				}
				cancel()
			}()

			RatisExploration(episodes, horizon, saveFile, ctx)

			close(doneCh)
		},
	}
}
