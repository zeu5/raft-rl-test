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
	case "Commit1WithRemainingRequests":
		machine = policies.NewRewardMachine(cbft.AtLeastHeight(2).And(types.RemainingRequests(15))) // .And(types.Rema))
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
		NumRequests:         20,
		CreateEmptyBlocks:   true,
	})
	colors := []cbft.CometColorFunc{cbft.ColorHRS(), cbft.ColorProposal(), cbft.ColorNumVotes(), cbft.ColorProposer()}

	partitionEnvConfig := types.PartitionEnvConfig{
		Painter:                cbft.NewCometStatePainter(colors...),
		Env:                    env,
		TicketBetweenPartition: 1,
		MaxMessagesPerTick:     100,
		StaySameStateUpto:      2,
		NumReplicas:            4,
		WithCrashes:            false,
		CrashLimit:             10,
		MaxInactive:            2,
		WithByzantine:          false,
		// Enables storing the time values in the partition environment
		RecordStats: false,
	}

	c := types.NewComparison(runs, saveFile, false)

	// Record the stats (time values) to the file
	c.AddEAnalysis(types.RecordPartitionStats(saveFile))
	c.AddAnalysis("crashes", cbft.CrashesAnalyzer(saveFile), types.NoopComparator())

	bugs := []types.BugDesc{
		{Name: "Round1", Check: cbft.ReachedRound1()},
		{Name: "DifferentProposers", Check: cbft.DifferentProposers()},
	}
	c.AddAnalysis("bugs", types.BugAnalyzer(saveFile, bugs...), types.BugComparator(saveFile))
	c.AddAnalysis("bugs_logs", cbft.BugLogRecorder(saveFile, bugs...), types.NoopComparator())
	// c.AddAnalysis("logs", cbft.RecordLogsAnalyzer(saveFile), types.NoopComparator())
	// c.AddAnalysis("states", cbft.RecordStateTraceAnalyzer(saveFile), types.NoopComparator())

	rm, ok := getCometPredicateHeirarchy(machine)
	if !ok {
		env.Cleanup()
		return
	}
	RMPolicy := policies.NewRewardMachinePolicy(rm, false)

	c.AddAnalysis("rm", policies.RewardMachineAnalyzer(RMPolicy), policies.RewardMachineCoverageComparator(saveFile, machine))

	c.AddExperiment(types.NewExperiment("RM", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      RMPolicy,
		Environment: types.NewPartitionEnv(partitionEnvConfig),
	}, types.RepConfigOff()))
	c.AddExperiment(types.NewExperiment("Random", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      types.NewRandomPolicy(),
		Environment: types.NewPartitionEnv(partitionEnvConfig),
	}, types.RepConfigOff()))
	strict := policies.NewStrictPolicy(types.NewRandomPolicy())
	strict.AddPolicy(policies.If(policies.Always()).Then(types.PickKeepSame()))

	c.AddExperiment(types.NewExperiment("Strict", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      strict,
		Environment: types.NewPartitionEnv(partitionEnvConfig),
	}, types.RepConfigOff()))

	c.RunWithCtx(ctx)
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
