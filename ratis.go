package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/ratis"
	"github.com/zeu5/raft-rl-test/types"
)

func RatisExploration(episodes, horizon int, saveFile string, ctx context.Context) {
	env := ratis.NewRatisRaftEnv(ctx, &ratis.RatisClusterConfig{
		NumNodes:            3,
		BasePort:            5000,
		BaseInterceptPort:   2023,
		InterceptListenAddr: "localhost:7074",
		// Todo: need to figure this out
		RatisJarPath: "",
	})
	colors := []ratis.RatisColorFunc{ratis.ColorState(), ratis.ColorCommit(), ratis.ColorLeader(), ratis.ColorVote(), ratis.ColorBoundedTerm(5)}

	partitionEnv := types.NewPartitionEnv(types.PartitionEnvConfig{
		Painter:                ratis.NewRatisStatePainter(colors...),
		Env:                    env,
		TicketBetweenPartition: 3,
		MaxMessagesPerTick:     3,
		StaySameStateUpto:      2,
		NumReplicas:            3,
	})

	c := types.NewComparison(runs)

	c.AddAnalysis("plot", ratis.CoverageAnalyzer(colors...), ratis.CoverageComparator(saveFile))
	c.AddAnalysis("bugs", ratis.BugAnalyzer(saveFile), ratis.BugComparator())

	c.AddExperiment(types.NewExperiment("NegReward", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      policies.NewSoftMaxNegFreqPolicy(0.1, 0.99, 1),
		Environment: partitionEnv,
	}))

	c.AddExperiment(types.NewExperiment("Random", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      types.NewRandomPolicy(),
		Environment: partitionEnv,
	}))

	c.AddExperiment(types.NewExperiment("BonusMax", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      policies.NewBonusPolicyGreedy(0.1, 0.99, 0.2),
		Environment: partitionEnv,
	}))

	c.Run()
	env.Cleanup()
}

func RatisExplorationCommand() *cobra.Command {
	return &cobra.Command{
		Use: "redisraft",
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
