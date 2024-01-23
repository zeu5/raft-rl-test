package benchmarks

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/raft"
	"github.com/zeu5/raft-rl-test/types"
	r "go.etcd.io/raft/v3"
)

func getPredHierEtcd(name string) (*policies.RewardMachine, bool, bool) {
	var machine *policies.RewardMachine = nil
	oneTime := false

	switch name {
	case "MultipleLeaderElections3":
		PredHierarchy_MultipleLeaderElections3 := policies.NewRewardMachine(raft.LeaderElectedPredicateNumber(3)) // final predicate - target space
		PredHierarchy_MultipleLeaderElections3.AddState(raft.LeaderElectedPredicateNumber(1), "OneElection")      // 1st step
		PredHierarchy_MultipleLeaderElections3.AddState(raft.LeaderElectedPredicateNumber(2), "TwoElections")     // 2nd ...
		machine = PredHierarchy_MultipleLeaderElections3
		oneTime = true
	case "OutdatedLog2":
		PredHierarchy_OutdatedLog2 := policies.NewRewardMachine(raft.LeaderElectedPredicateNumber(2).And(raft.AtLeastOneLogEmpty().And(raft.AtLeastOneLogNotEmpty()).And(raft.StackSizeLowerBound(1)))) // final predicate - target space
		PredHierarchy_OutdatedLog2.AddState(raft.LeaderElectedPredicateNumber(1).And(raft.AtLeastOneLogEmpty().And(raft.StackSizeLowerBound(1))), "OneElection")                                        // 1st step
		PredHierarchy_OutdatedLog2.AddState(raft.LeaderElectedPredicateNumber(1).And(raft.AtLeastOneLogNotEmpty().And(raft.AtLeastOneLogEmpty()).And(raft.StackSizeLowerBound(1))), "TwoElections")     // 2nd ...
		machine = PredHierarchy_OutdatedLog2
		oneTime = true
	case "OutdatedLogSpec":
		PredHierarchy_OutdatedLogSpec := policies.NewRewardMachine(raft.LeaderElectedPredicateNumberWithTerms(2, []uint64{2, 4}).And(raft.AtLeastOneLogEmpty().And(raft.AtLeastOneLogNotEmptyTerm(2)).And(raft.StackSizeLowerBound(1)))) // final predicate - target space
		machine = PredHierarchy_OutdatedLogSpec
		oneTime = true
	case "Term1":
		machine = policies.NewRewardMachine(raft.AllInTerm(3))
		oneTime = false
	}

	return machine, oneTime, machine != nil
}

func EtcdRaftBugs(episodes, horizon int, savePath string, ctx context.Context) {
	// config of the running system
	// requests = 3
	raftConfig := raft.RaftEnvironmentConfig{
		Replicas:          3,
		ElectionTick:      7, // lower bound for a process to try to go to new term (starting an election) - double of this is upperbound
		HeartbeatTick:     2, // frequency of heartbeats
		Timeouts:          timeouts,
		Requests:          3,
		SnapshotFrequency: 0,
	}

	rlConfig := raft.RLConfig{
		TicksBetweenPartition: 3,
		MaxMessagesPerTick:    100,
		StaySameStateUpTo:     5,
		WithCrashes:           true,
		CrashLimit:            10,
	}

	// abstraction for both plot and RL
	availableColors := make(map[string]raft.RaftColorFunc)
	availableColors["state"] = raft.ColorState()                 // replica internal state
	availableColors["commit"] = raft.ColorCommit()               // number of committed entries? includes config changes, leader elect, request entry
	availableColors["leader"] = raft.ColorLeader()               // if a replica is leader? boolean?
	availableColors["vote"] = raft.ColorVote()                   // ?
	availableColors["boundedTerm5"] = raft.ColorBoundedTerm(5)   // current term, bounded to the passed value
	availableColors["boundedTerm10"] = raft.ColorBoundedTerm(10) // current term, bounded to the passed value
	availableColors["applied"] = raft.ColorApplied()
	availableColors["snapshotIndex"] = raft.ColorSnapshotIndex()
	availableColors["snapshotTerm"] = raft.ColorSnapshotTerm()
	availableColors["replicaID"] = raft.ColorReplicaID()
	availableColors["logLength"] = raft.ColorLogLength()
	availableColors["log"] = raft.ColorLog()

	chosenColors := []string{
		"state",
		"commit",
		"leader",
		"vote",
		"boundedTerm10",
		"logLength",
		// "replicaID",
	}

	// colors is one abstraction definition
	// colors := []raft.RaftColorFunc{
	// 	raft.ColorState(),
	// 	raft.ColorCommit(),
	// 	raft.ColorLeader(),
	// 	raft.ColorVote(),
	// 	raft.ColorBoundedTerm(10),
	// }

	colors := make([]raft.RaftColorFunc, 0)
	for _, color := range chosenColors {
		colors = append(colors, availableColors[color])
	}

	// build a machine with sequence of predicates
	// elect a leader, commit with one replica not updated, elect that replica as leader

	PredHierarchy_MultipleLeaderElections := policies.NewRewardMachine(raft.LeaderElectedPredicateNumberWithTerms(3, []uint64{2, 3, 4})) // final predicate - target space
	PredHierarchy_MultipleLeaderElections.AddState(raft.LeaderElectedPredicateStateWithTerm(2), "FirstElection")                         // 1st step
	PredHierarchy_MultipleLeaderElections.AddState(raft.LeaderElectedPredicateNumberWithTerms(1, []uint64{2}), "FirstElectionCommit")
	PredHierarchy_MultipleLeaderElections.AddState(raft.LeaderElectedPredicateStateWithTerm(3).And(raft.LeaderElectedPredicateNumberWithTerms(1, []uint64{2})), "SecondElection") // 1st step
	PredHierarchy_MultipleLeaderElections.AddState(raft.LeaderElectedPredicateNumberWithTerms(2, []uint64{2, 3}), "SecondElectionCommit")
	PredHierarchy_MultipleLeaderElections.AddState(raft.LeaderElectedPredicateStateWithTerm(4).And(raft.LeaderElectedPredicateNumberWithTerms(2, []uint64{2, 3})), "ThirdElection") // 1st step

	// elect a leader, commit a request and then elect another leader
	PredHierarchy_EntriesInDifferentTermsInLogNoStack := policies.NewRewardMachine(raft.EntriesInDifferentTermsInLog(2))
	PredHierarchy_EntriesInDifferentTermsInLogNoStack.AddState(raft.LeaderElectedPredicateNumber(1), "LeaderElected")
	PredHierarchy_EntriesInDifferentTermsInLogNoStack.AddState(raft.AtLeastOneLogOneEntry(), "OneEntryLog")
	PredHierarchy_EntriesInDifferentTermsInLogNoStack.AddState(raft.AtLeastOneLogOneEntryPlusReplicaInHigherTerm(), "OneEntryLogAndReplicaInHigherTerm")
	PredHierarchy_EntriesInDifferentTermsInLogNoStack.AddState(raft.AtLeastOneLogOneEntryPlusSubsequentLeaderElection(), "OneEntryLogAndSubsequentLeaderElection")

	// elect a leader and commit a request
	PredHierarchy_EntriesInDifferentTermsInLog := policies.NewRewardMachine(raft.EntriesInDifferentTermsInLog(2).And(raft.HighestTermForReplicas(6)))
	PredHierarchy_EntriesInDifferentTermsInLog.AddState(raft.LeaderElectedPredicateState().And(raft.HighestTermForReplicas(2)), "LeaderElected AND HTerm2")
	PredHierarchy_EntriesInDifferentTermsInLog.AddState(raft.AtLeastOneLogNotEmpty().And(raft.StackSizeLowerBound(2)).And(raft.HighestTermForReplicas(2)), "EntryCommitted AND TwoAvailableRequests AND HTerm2")
	PredHierarchy_EntriesInDifferentTermsInLog.AddState(raft.AtLeastOneEntryANDSubsequentLeaderElection().And(raft.StackSizeLowerBound(2)).And(raft.HighestTermForReplicas(4)), "OneEntryLogAndSubsequentLeaderElection AND TwoAvailableRequests AND HTerm2")

	// elect a leader and commit a request
	PredHierarchy_4 := policies.NewRewardMachine(raft.EntriesInDifferentTermsInLog(2))
	PredHierarchy_4.AddState(raft.LeaderElectedPredicateSpecific(1), "LeaderElected(1)")
	PredHierarchy_4.AddState(raft.LeaderElectedPredicateSpecific(1).And(raft.ExactEntriesInLogSpecific(1, 1)), "CommitOneEntry(1)")
	PredHierarchy_4.AddState(raft.LeaderElectedPredicateSpecific(2).And(raft.ExactEntriesInLogSpecific(1, 1)), "ChangeOfLeader(1->2)")

	PredHierarchy_5 := policies.NewRewardMachine(raft.EntriesInDifferentTermsInLog(2))
	PredHierarchy_5.AddState(raft.ExactEntriesInLog(1), "OneEntryCommitted")
	PredHierarchy_5.AddState(raft.InStateWithCommittedEntries(r.StateCandidate, 1), "CandidateWithCommittedEntry")
	PredHierarchy_5.AddState(raft.InStateWithCommittedEntries(r.StateLeader, 1), "NoPartitionWithCandidate")

	// PredHierarchy_6 := policies.NewRewardMachine(raft.EntriesInDifferentTermsInLog(2))

	PredHier, oneTime, _ := getPredHierEtcd(hierarchy)
	PredHierName := hierarchy
	PHPolicy := policies.NewRewardMachinePolicy(PredHier, oneTime)

	// c is general experiment
	// colors ... , expanded list, can omit the argument
	// Analyzer takes the path to save data and colors... is the abstraction used to plot => makes the datasets
	// PlotComparator => makes plots from data
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

	// here you add different traces analysis and comparators -- to process traces into a dataset (analyzer) and output the results (comparator)
	c.AddAnalysis("Plot", raft.NewRaftAnalyzer(saveFile, colors...), raft.RaftPlotComparator(saveFile))
	c.AddAnalysis("Bugs", types.NewBugAnalyzer(
		path.Join(saveFile, "bugs"),
		types.BugDesc{Name: "MultipleLeaders", Check: raft.MultipleLeaders()},
		types.BugDesc{Name: "ReducedLog", Check: raft.ReducedLog()},
		types.BugDesc{Name: "ModifiedLog", Check: raft.ModifiedLog()},
		types.BugDesc{Name: "InconsistentLogs", Check: raft.InconsistentLogs()},
		// types.BugDesc{Name: "DummyBug2", Check: raft.DummyBug2()},
	), types.BugComparator(saveFile))

	// c.AddAnalysis("CommitOnlyOneEntry", policies.RewardMachineAnalyzer(PredHierarchy_3), policies.RewardMachineCoverageComparator(saveFile))
	c.AddAnalysis(PredHierName, policies.NewRewardMachineAnalyzer(PHPolicy), policies.RewardMachineCoverageComparator(saveFile, PredHierName))
	// c.AddAnalysis("PrintReadable", raft.RaftReadableAnalyzer(savePath), raft.RaftEmptyComparator())

	// here you add different policies with their parameters
	// c.AddExperiment(types.NewExperiment("RL", &types.AgentConfig{
	// 	Episodes:    episodes,
	// 	Horizon:     horizon,
	// 	Policy:      policies.NewSoftMaxNegFreqPolicy(0.3, 0.7, 1),
	// 	Environment: ggetRaftPartEnvCfg(raftConfig, colors, rlConfig),
	// }))
	c.AddExperiment(types.NewExperiment(
		"Random",
		types.NewRandomPolicy(),
		getRaftPartEnvCfg(raftConfig, colors, rlConfig),
	))
	c.AddExperiment(types.NewExperiment(
		"BonusMaxRL",
		policies.NewBonusPolicyGreedy(0.1, 0.95, 0.05),
		getRaftPartEnvCfg(raftConfig, colors, rlConfig),
	))
	// c.AddExperiment(types.NewExperiment("PredHierarchy_1", &types.AgentConfig{
	// 	Episodes:    episodes,
	// 	Horizon:     horizon,
	// 	Policy:      policies.NewRewardMachinePolicy(PredHierarchy_1),
	// 	Environment: getRaftPartEnv(raftConfig, colors),
	// }))

	// strictPolicy := policies.NewStrictPolicy(types.NewRandomPolicy())
	// strictPolicy.AddPolicy(policies.If(policies.Always()).Then(types.PickKeepSame()))
	// c.AddExperiment(types.NewExperiment("Strict", &types.AgentConfig{
	// 	Episodes:    episodes,
	// 	Horizon:     horizon,
	// 	Policy:      policies.NewStrictPolicy(strictPolicy),
	// 	Environment: getRaftPartEnvCfg(raftConfig, colors, rlConfig),
	// }))
	c.AddExperiment(types.NewExperiment(
		PredHierName,
		PHPolicy,
		getRaftPartEnvCfg(raftConfig, colors, rlConfig),
	))
	// c.AddExperiment(types.NewExperiment("PredHierarchy_3", &types.AgentConfig{
	// 	Episodes:    episodes,
	// 	Horizon:     horizon,
	// 	Policy:      policies.NewRewardMachinePolicy(PredHierarchy_3),
	// 	Environment: getRaftPartEnvCfg(raftConfig, colors, rlConfig),
	// }))

	// print experiment configuration to terminal
	fmt.Print(raftConfig.String())
	fmt.Print(rlConfig.String())
	fmt.Print(PrintColors(chosenColors))

	c.Run(ctx)
}

func EtcdRaftBugsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "etcd-raft-bugs",
		Run: func(cmd *cobra.Command, args []string) {
			EtcdRaftBugs(episodes, horizon, saveFile, context.Background())
		},
	}
	cmd.PersistentFlags().IntVarP(&requests, "requests", "r", 3, "Number of requests to run with")
	cmd.PersistentFlags().StringVarP(&hierarchy, "phierarchy", "p", "none", "Specify the predicate hierarchy to use")
	return cmd
}

// print colors chosen for abstraction
func PrintColors(colors []string) string {
	result := "Applied Colors: \n"
	for _, color := range colors {
		result = fmt.Sprintf("%s %s\n", result, color)
	}
	result = fmt.Sprintf("%s\n", result)

	return result
}
