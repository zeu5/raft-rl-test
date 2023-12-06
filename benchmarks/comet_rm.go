package benchmarks

import (
	"context"
	"os"
	"os/signal"
	"path"

	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/cbft"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/types"
)

func getCometPredicateHeirarchy(name string) (*policies.RewardMachine, bool) {
	var machine *policies.RewardMachine = nil
	switch name {
	case "AnyRound1AfterCommit":
		machine = policies.NewRewardMachine(cbft.AllAtLeastRound(1).And(cbft.AtLeastHeight(2)))
		machine.AddState(cbft.AnyAtHeight(2), "Height2")
	case "AnyRound1":
		machine = policies.NewRewardMachine(cbft.AnyReachedRound(1))
	case "AllRound1":
		machine = policies.NewRewardMachine(cbft.AllAtLeastRound(1))
	case "AnyLocked":
		machine = policies.NewRewardMachine(cbft.EmptyLockedForAll().Not())
	case "AllLocked":
		machine = policies.NewRewardMachine(cbft.LockedForAll())
	case "DifferentLocked":
		machine = policies.NewRewardMachine(cbft.DifferentLocked())
	case "SameLocked":
		machine = policies.NewRewardMachine(cbft.SameLockedForAll())
	case "Round1NoLocked":
		machine = policies.NewRewardMachine(cbft.EmptyLockedForAll().And(cbft.AnyReachedRound(1)))
		machine.AddState(cbft.AnyReachedRound(1), "Round1")
		machine.AddState(cbft.EmptyLockedForAll(), "No Locked")
	case "Round1Locked":
		machine = policies.NewRewardMachine(cbft.EmptyLockedForAll().Not().And(cbft.AnyReachedRound(1)))
		machine.AddState(cbft.EmptyLockedForAll().Not(), "AnyLocked")
		machine.AddState(cbft.AnyReachedRound(1), "Round1")
	case "Commit1":
		machine = policies.NewRewardMachine(cbft.AnyAtHeight(2))
	case "StepPreCommit":
		machine = policies.NewRewardMachine(cbft.AnyReachedStep("RoundStepPrecommit"))
	case "StepPrevote":
		machine = policies.NewRewardMachine(cbft.AnyReachedStep("RoundStepPrevote"))
	}
	return machine, machine != nil
}

func CometRM(machine string, episodes, horizon int, saveFile string, ctx context.Context) {
	env := cbft.NewCometEnv(ctx, &cbft.CometClusterConfig{
		CometBinaryPath:     "/home/snagendra/go/src/github.com/zeu5/cometbft/build/cometbft",
		InterceptListenPort: 7074,
		BaseRPCPort:         26756,
		BaseWorkingDir:      path.Join("/home/snagendra/go/src/github.com/zeu5/raft-rl-test", saveFile, "tmp"),
		NumNodes:            4,
		NumRequests:         2,
	})
	colors := []cbft.CometColorFunc{cbft.ColorHRS(), cbft.ColorProposal(), cbft.ColorNumVotes(), cbft.ColorProposer()}

	partitionEnvConfig := types.PartitionEnvConfig{
		Painter:                cbft.NewCometStatePainter(colors...),
		Env:                    env,
		TicketBetweenPartition: 3,
		MaxMessagesPerTick:     20,
		StaySameStateUpto:      2,
		NumReplicas:            4,
		WithCrashes:            false,
		CrashLimit:             10,
		MaxInactive:            2,
		WithByzantine:          false,
	}

	c := types.NewComparison(runs, saveFile, false)

	// c.AddEAnalysis(types.RecordPartitionStats(saveFile))
	c.AddAnalysis("crashes", cbft.CrashesAnalyzer(saveFile), types.NoopComparator())
	// c.AddAnalysis("logs", cbft.RecordLogsAnalyzer(saveFile), types.NoopComparator())
	c.AddAnalysis("states", cbft.RecordStateTraceAnalyzer(saveFile), types.NoopComparator())

	rm, ok := getCometPredicateHeirarchy(machine)
	if !ok {
		env.Cleanup()
		return
	}
	RMPolicy := policies.NewRewardMachinePolicy(rm, false)

	c.AddAnalysis("rm", policies.RewardMachineAnalyzer(RMPolicy), policies.RewardMachineCoverageComparator(saveFile))

	c.AddExperiment(types.NewExperiment("RM", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      RMPolicy,
		Environment: types.NewPartitionEnv(partitionEnvConfig),
	}))
	c.AddExperiment(types.NewExperiment("Random", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      types.NewRandomPolicy(),
		Environment: types.NewPartitionEnv(partitionEnvConfig),
	}))

	c.Run()
	env.Cleanup()
}

func CometRMCommand() *cobra.Command {
	return &cobra.Command{
		Use:  "comet-rm [machine]",
		Args: cobra.ExactArgs(1),
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

			CometRM(args[0], episodes, horizon, saveFile, ctx)

			close(doneCh)
		},
	}
}
