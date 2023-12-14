package benchmarks

import (
	"context"
	"os"
	"os/signal"
	"path"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/cbft"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/types"
)

func CometExploration(episodes, horizon int, saveFile string, ctx context.Context) {

	exec, _ := os.Executable()
	curDir := filepath.Dir(exec)

	env := cbft.NewCometEnv(ctx, &cbft.CometClusterConfig{
		CometBinaryPath:     path.Join(curDir, "cbft", "cometbft"),
		InterceptListenPort: 7074,
		BaseRPCPort:         26756,
		BaseWorkingDir:      path.Join(curDir, saveFile, "tmp"),
		NumNodes:            4,
		NumRequests:         2,
	})
	colors := []cbft.CometColorFunc{cbft.ColorHRS(), cbft.ColorProposal(), cbft.ColorNumVotes(), cbft.ColorProposer()}

	partitionEnv := types.NewPartitionEnv(types.PartitionEnvConfig{
		Painter:                cbft.NewCometStatePainter(colors...),
		Env:                    env,
		TicketBetweenPartition: 3,
		MaxMessagesPerTick:     20,
		StaySameStateUpto:      2,
		NumReplicas:            4,
		WithCrashes:            true,
		CrashLimit:             10,
		MaxInactive:            2,
		WithByzantine:          true,
		MaxByzantine:           1,
	})

	c := types.NewComparison(runs, saveFile, false)

	c.AddAnalysis("plot", cbft.CoverageAnalyzer(colors...), cbft.CoverageComparator(saveFile))
	// c.AddAnalysis("logs", cbft.RecordLogsAnalyzer(saveFile), types.NoopComparator())
	// c.AddAnalysis("state_trace", cbft.RecordStateTraceAnalyzer(saveFile), types.NoopComparator())
	c.AddAnalysis("crashes", cbft.CrashesAnalyzer(saveFile), types.NoopComparator())
	c.AddAnalysis("bugs", types.BugAnalyzer(saveFile,
		types.BugDesc{Name: "Round1", Check: cbft.ReachedRound1()},
		types.BugDesc{Name: "DifferentProposers", Check: cbft.DifferentProposers()},
	), types.BugComparator(saveFile))

	c.AddExperiment(types.NewExperiment("NegReward", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      types.NewSoftMaxNegPolicy(0.1, 0.99, 1),
		Environment: partitionEnv,
	}))

	c.AddExperiment(types.NewExperiment("Random", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      types.NewRandomPolicy(),
		Environment: partitionEnv,
	}))

	// strict := policies.NewStrictPolicy(types.NewRandomPolicy())
	// strict.AddPolicy(policies.If(policies.Always()).Then(types.PickKeepSame()))

	// c.AddExperiment(types.NewExperiment("Strict", &types.AgentConfig{
	// 	Episodes:    episodes,
	// 	Horizon:     horizon,
	// 	Policy:      strict,
	// 	Environment: partitionEnv,
	// }))

	c.AddExperiment(types.NewExperiment("BonusMax", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      policies.NewBonusPolicyGreedy(0.1, 0.99, 0.2),
		Environment: partitionEnv,
	}))

	c.Run()
	env.Cleanup()
}

func CometCommand() *cobra.Command {
	return &cobra.Command{
		Use: "comet",
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

			CometExploration(episodes, horizon, saveFile, ctx)

			close(doneCh)
		},
	}
}
