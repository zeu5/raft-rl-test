package benchmarks

import (
	"context"
	"os"
	"os/signal"
	"path"
	"time"

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
		NumRequests:           5,
		RaftModulePath:        "/home/snagendra/go/src/github.com/zeu5/raft-rl-test/redisraft/redisraft.so",
		RedisServerBinaryPath: "/home/snagendra/go/src/github.com/zeu5/raft-rl-test/redisraft/redis-server",

		RequestTimeout:  100,
		ElectionTimeout: 400,

		TickLength: 25,
	}, path.Join(saveFile, "tickLength"))

	colors := []redisraft.RedisRaftColorFunc{redisraft.ColorState(), redisraft.ColorCommit(), redisraft.ColorLeader(), redisraft.ColorVote(), redisraft.ColorBoundedTerm(5), redisraft.ColorIndex(), redisraft.ColorBoundedLog(5)}

	partitionEnv := types.NewPartitionEnv(types.PartitionEnvConfig{
		Painter:                redisraft.NewRedisRaftStatePainter(colors...),
		Env:                    env,
		TicketBetweenPartition: 4,
		MaxMessagesPerTick:     100,
		StaySameStateUpto:      5,
		NumReplicas:            3,
		WithCrashes:            true,
		CrashLimit:             3,
		MaxInactive:            1,

		TerminalPredicate: redisraft.MaxTerm(3),
	})

	c := types.NewComparison(&types.ComparisonConfig{
		Runs:       runs,
		Episodes:   episodes,
		Horizon:    horizon,
		RecordPath: saveFile,
		Timeout:    10 * time.Second,
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

	c.AddAnalysis("plot", redisraft.NewCoverageAnalyzer(horizon, colors...), redisraft.CoverageComparator(saveFile, horizon))
	// c.AddAnalysis("bugs", redisraft.NewBugAnalyzer(saveFile), redisraft.BugComparator())

	c.AddExperiment(types.NewExperiment(
		"NegReward",
		policies.NewSoftMaxNegFreqPolicy(0.1, 0.99, 1),
		partitionEnv,
	))

	c.AddExperiment(types.NewExperiment(
		"Random",
		types.NewRandomPolicy(),
		partitionEnv,
	))

	c.AddExperiment(types.NewExperiment(
		"BonusMax",
		policies.NewBonusPolicyGreedy(0.1, 0.99, 0.05),
		partitionEnv,
	))

	c.Run(ctx)
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
