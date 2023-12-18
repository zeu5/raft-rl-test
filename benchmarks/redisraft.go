package benchmarks

import (
	"context"
	"os"
	"os/signal"
	"path"

	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/redisraft"
	"github.com/zeu5/raft-rl-test/types"
)

func RedisRaftExploration(episodes, horizon int, saveFile string, ctx context.Context) {
	env := redisraft.NewRedisRaftEnv(ctx, &redisraft.ClusterConfig{
		NumNodes:              3,
		BasePort:              5000,
		BaseInterceptPort:     2023,
		ID:                    1,
		InterceptListenAddr:   "localhost:7074",
		WorkingDir:            path.Join(saveFile, "tmp"),
		NumRequests:           3,
		RaftModulePath:        "/home/snagendra/go/src/github.com/zeu5/raft-rl-test/redisraft/redisraft.so",
		RedisServerBinaryPath: "/home/snagendra/go/src/github.com/zeu5/raft-rl-test/redisraft/redis-server",
	})
	colors := []redisraft.RedisRaftColorFunc{redisraft.ColorState(), redisraft.ColorCommit(), redisraft.ColorLeader(), redisraft.ColorVote(), redisraft.ColorBoundedTerm(5)}

	partitionEnv := types.NewPartitionEnv(types.PartitionEnvConfig{
		Painter:                redisraft.NewRedisRaftStatePainter(colors...),
		Env:                    env,
		TicketBetweenPartition: 3,
		MaxMessagesPerTick:     3,
		StaySameStateUpto:      2,
		NumReplicas:            3,
		WithCrashes:            true,
	})

	c := types.NewComparison(runs, saveFile, false)

	c.AddAnalysis("plot", redisraft.CoverageAnalyzer(colors...), redisraft.CoverageComparator(saveFile))
	c.AddAnalysis("bugs", redisraft.BugAnalyzer(saveFile), redisraft.BugComparator())

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

	c.RunWithCtx(ctx)
	env.Cleanup()
}

func RedisRaftCommand() *cobra.Command {
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

			RedisRaftExploration(episodes, horizon, saveFile, ctx)

			close(doneCh)
		},
	}
}
