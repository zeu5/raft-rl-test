package main

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

func RedisRaftRM(episodes, horizon int, saveFile string, ctx context.Context) {
	env := redisraft.NewRedisRaftEnv(ctx, &redisraft.ClusterConfig{
		NumNodes:            3,
		BasePort:            5000,
		BaseInterceptPort:   2023,
		ID:                  1,
		InterceptListenAddr: "localhost:7074",
		WorkingDir:          path.Join(saveFile, "tmp"),
		NumRequests:         3,
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

	c := types.NewComparison(runs)

	// c.AddAnalysis("plot", redisraft.CoverageAnalyzer(colors...), redisraft.CoverageComparator(saveFile))
	c.AddAnalysis("bugs", redisraft.BugAnalyzer(saveFile), redisraft.BugComparator())

	// rm := policies.NewRewardMachine(redisraft.LeaderElected()) // This is always true. The system starts with a config where the leader is elected
	rm := policies.NewRewardMachine(redisraft.TermNumber(2))
	// rm := policies.NewRewardMachine(redisraft.OnlyFollowersAndLeader())

	c.AddAnalysis("rm", policies.RewardMachineAnalyzer(rm), policies.RewardMachineCoverageComparator(saveFile))

	c.AddExperiment(types.NewExperiment("RM", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      policies.NewRewardMachinePolicy(rm),
		Environment: partitionEnv,
	}))

	c.Run()
	env.Cleanup()
}

func RedisRaftRMCommand() *cobra.Command {
	return &cobra.Command{
		Use: "redisraft-rm",
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

			RedisRaftRM(episodes, horizon, saveFile, ctx)

			close(doneCh)
		},
	}
}
