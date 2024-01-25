package benchmarks

import (
	"context"
	"os"
	"os/signal"
	"path"
	"time"

	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/cbft"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/types"
)

func getCometPredicateHeirarchy(name string) (*policies.RewardMachine, bool, bool) {
	var machine *policies.RewardMachine = nil
	onetime := false
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
		machine = policies.NewRewardMachine(cbft.AnyAtLeastHeight(2))
	case "StepPreCommit":
		machine = policies.NewRewardMachine(cbft.AllInRound(0).And(cbft.AnyReachedStep("RoundStepPrecommit")))
		onetime = true
	case "StepPrevote":
		machine = policies.NewRewardMachine(cbft.AnyReachedStep("RoundStepPrevote"))
	case "KeepQuorumCommit":
		machine = policies.NewRewardMachine(cbft.AnyAtHeight(2))
		machine.AddState(types.NSamePartitionActive(3), "KeepQuorum")
	case "CommitWithPreCommit":
		machine = policies.NewRewardMachine(cbft.AnyAtHeight(2))
		machine.AddState(cbft.AnyReachedStep("RoundStepPrecommit"), "Precommit")
	case "Commit1WithRemainingRequests":
		machine = policies.NewRewardMachine(cbft.AtLeastHeight(2).And(types.RemainingRequests(15))) // .And(types.Rema))
	}
	return machine, onetime, machine != nil
}

func allPredicatesNames() []string {
	return []string{
		"AnyRound1AfterCommit",
		"AnyRound1",
		"AllRound1",
		"AnyLocked",
		"AllLocked",
		"DifferentLocked",
		"SameLocked",
		"Round1NoLocked",
		"Round1Locked",
		"Commit1",
		"StepPreCommit",
		"StepPrevote",
		"KeepQuorumCommit",
		"CommitWithPreCommit",
		"Commit1WithRemainingRequests",
	}
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
		TickDuration:        50 * time.Millisecond,
	})
	colors := []cbft.CometColorFunc{cbft.ColorHeight(), cbft.ColorStep(), cbft.ColorProposal(), cbft.ColorCurRoundVotes(), cbft.ColorRound()}

	partitionEnvConfig := types.PartitionEnvConfig{
		Painter:                cbft.NewCometStatePainter(colors...),
		Env:                    env,
		TicketBetweenPartition: 3,
		MaxMessagesPerTick:     100,
		StaySameStateUpto:      2,
		NumReplicas:            4,
		WithCrashes:            false,
		CrashLimit:             10,
		MaxInactive:            2,
		WithByzantine:          false,
		// Enables storing the time values in the partition environment
		RecordStats: true,
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
		ReportConfig: types.RepConfigStandard(),
	})

	// Record the stats (time values) to the file
	c.AddAnalysis("crashes", cbft.NewCrashesAnalyzer(saveFile), types.NoopComparator())

	// bugs := []types.BugDesc{
	// 	// {Name: "Round1", Check: cbft.ReachedRound1()},
	// 	{Name: "DifferentProposers", Check: cbft.DifferentProposers()},
	// }
	// c.AddAnalysis("bugs", types.NewBugAnalyzer(saveFile, bugs...), types.BugComparator(saveFile))
	// c.AddAnalysis("bugs_logs", cbft.NewBugLogRecorder(saveFile, bugs...), types.NoopComparator())
	// c.AddAnalysis("logs", cbft.RecordLogsAnalyzer(saveFile), types.NoopComparator())
	// c.AddAnalysis("states", cbft.RecordStateTraceAnalyzer(saveFile), types.NoopComparator())

	if machine == "all" {
		for _, heirarchyName := range allPredicatesNames() {
			machine, onetime, ok := getCometPredicateHeirarchy(heirarchyName)
			if !ok {
				continue
			}
			policy := policies.NewRewardMachinePolicy(machine, onetime)
			c.AddExperiment(types.NewExperiment(heirarchyName, policy, types.NewPartitionEnv(partitionEnvConfig)))
			c.AddAnalysis(heirarchyName, policies.NewRewardMachineAnalyzer(policy), policies.RewardMachineCoverageComparator(saveFile, heirarchyName))
		}
	} else {
		rm, onetime, ok := getCometPredicateHeirarchy(machine)
		if !ok {
			env.Cleanup()
			return
		}
		RMPolicy := policies.NewRewardMachinePolicy(rm, onetime)

		c.AddAnalysis("rm", policies.NewRewardMachineAnalyzer(RMPolicy), policies.RewardMachineCoverageComparator(saveFile, machine))
		c.AddExperiment(types.NewExperiment(
			"RM",
			RMPolicy,
			types.NewPartitionEnv(partitionEnvConfig),
		))
	}

	c.AddExperiment(types.NewExperiment(
		"Random",
		types.NewRandomPolicy(),
		types.NewPartitionEnv(partitionEnvConfig),
	))
	c.AddExperiment(types.NewExperiment(
		"BonusMax",
		policies.NewBonusPolicyGreedy(0.1, 0.99, 0.2),
		types.NewPartitionEnv(partitionEnvConfig),
	))
	// strict := policies.NewStrictPolicy(types.NewRandomPolicy())
	// strict.AddPolicy(policies.If(policies.Always()).Then(types.PickKeepSame()))

	// c.AddExperiment(types.NewExperiment(
	// 	"Strict",
	// 	strict,
	// 	types.NewPartitionEnv(partitionEnvConfig),
	// ))

	c.Run(ctx)
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
