package main

import (
	"fmt"
	"path"

	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/raft"
	"github.com/zeu5/raft-rl-test/types"
	r "go.etcd.io/raft/v3"
)

// func getRewardMachine(name string) (*policies.RewardMachine, bool) {
// 	var machine *policies.RewardMachine = nil
// 	switch name {
// 	case "InStatePrimary":
// 		machine = policies.NewRewardMachine(rsl.InState(rsl.StateStablePrimary))
// 	case "SinglePrimary":
// 		machine = policies.NewRewardMachine(rsl.NodePrimary(1))
// 	case "NumDecided":
// 		machine = policies.NewRewardMachine(rsl.NumDecided(2))
// 	case "ChangePrimary":
// 		m2 := policies.NewRewardMachine(rsl.NodePrimary(2))
// 		m2.AddState(rsl.NodePrimary(1), "OnePrimary")
// 		machine = m2
// 	case "NodeDecidedAndPrimary":
// 		machine = policies.NewRewardMachine(rsl.NodeNumDecided(1, 2).And(rsl.NodePrimary(1)))
// 	case "NodeDecidedAfterPrimary":
// 		m4 := policies.NewRewardMachine(rsl.NodeNumDecided(1, 2))
// 		m4.AddState(rsl.NodePrimary(1).And(rsl.NumDecided(0)), "NodeOnePrimary")
// 		machine = m4
// 	case "InBallot":
// 		machine = policies.NewRewardMachine(rsl.InBallot(2))
// 	case "NodeInBallot":
// 		machine = policies.NewRewardMachine(rsl.NodeInBallot(1, 2))
// 	case "InPreparedBallot":
// 		machine = policies.NewRewardMachine(rsl.InPreparedBallot(2))
// 	case "NodeInPreparedBallot":
// 		machine = policies.NewRewardMachine(rsl.NodeInPreparedBallot(1, 2))
// 	case "DifferentBallotCommits":
// 		machine = policies.NewRewardMachine(rsl.AtLeastDecided(2))
// 		machine.AddState(rsl.InStateAndBallot(rsl.StateStablePrimary, 1), "Ballot1Primary")
// 		machine.AddState(rsl.AtMostDecided(1), "AtMost1Decided")
// 		machine.AddState(rsl.InStateAndBallot(rsl.StateStablePrimary, 2), "Ballot2Primary")
// 	case "OutOfSync":
// 		machine = policies.NewRewardMachine(rsl.AllInSync().And(rsl.AllAtLeastBallot(3)))
// 		machine.AddState(rsl.OutSyncBallotBy(2).And(rsl.AllAtMostBallot(3)), "OutOfSync")
// 		// case "CrashAndHeal":
// 		// 	machine = policies.NewRewardMachine()
// 	}
// 	return machine, machine != nil
// }

func EtcdRaftBugs(episodes, horizon int, savePath string) {
	// config of the running system
	// requests = 3
	raftConfig := raft.RaftEnvironmentConfig{
		Replicas:      3,
		ElectionTick:  7, // lower bound for a process to try to go to new term (starting an election) - double of this is upperbound
		HeartbeatTick: 2, // frequency of heartbeats
		Timeouts:      timeouts,
		Requests:      3,
	}

	rlConfig := raft.RLConfig{
		TicksBetweenPartition: 3,
		MaxMessagesPerTick:    100,
		StaySameStateUpTo:     5,
		WithCrashes:           false,
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
	PredHierarchy_MultipleLeaderElections3 := policies.NewRewardMachine(raft.LeaderElectedPredicateNumber(3)) // final predicate - target space
	PredHierarchy_MultipleLeaderElections3.AddState(raft.LeaderElectedPredicateNumber(1), "OneElection")      // 1st step
	PredHierarchy_MultipleLeaderElections3.AddState(raft.LeaderElectedPredicateNumber(2), "TwoElections")     // 2nd ...

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

	RMPolicyName := "PredHierarchy_EntriesInDifferentTermsInLog"
	RMPolicy := policies.NewRewardMachinePolicy(PredHierarchy_EntriesInDifferentTermsInLog, true)

	// c is general experiment
	// colors ... , expanded list, can omit the argument
	// Analyzer takes the path to save data and colors... is the abstraction used to plot => makes the datasets
	// PlotComparator => makes plots from data
	c := types.NewComparison(runs)

	// here you add different traces analysis and comparators -- to process traces into a dataset (analyzer) and output the results (comparator)
	// c.AddAnalysis("Plot", raft.RaftAnalyzer(saveFile, colors...), raft.RaftPlotComparator(saveFile))
	c.AddAnalysis("Bugs", types.BugAnalyzer(
		path.Join(saveFile, "bugs"),
		types.BugDesc{Name: "MultipleLeaders", Check: raft.MultipleLeaders()},
		types.BugDesc{Name: "ReducedLog", Check: raft.ReducedLog()},
		types.BugDesc{Name: "ModifiedLog", Check: raft.ModifiedLog()},
		types.BugDesc{Name: "InconsistentLogs", Check: raft.InconsistentLogs()},
		types.BugDesc{Name: "DummyBug2", Check: raft.DummyBug2()},
	), types.BugComparator(saveFile))

	// c.AddAnalysis("CommitOnlyOneEntry", policies.RewardMachineAnalyzer(PredHierarchy_3), policies.RewardMachineCoverageComparator(saveFile))
	c.AddAnalysis("CommitEntriesInDifferentTerms", policies.RewardMachineAnalyzer(RMPolicy), policies.RewardMachineCoverageComparator(saveFile))
	// c.AddAnalysis("PrintReadable", raft.RaftReadableAnalyzer(savePath), raft.RaftEmptyComparator())

	// here you add different policies with their parameters
	// c.AddExperiment(types.NewExperiment("RL", &types.AgentConfig{
	// 	Episodes:    episodes,
	// 	Horizon:     horizon,
	// 	Policy:      policies.NewSoftMaxNegFreqPolicy(0.3, 0.7, 1),
	// 	Environment: getRaftPartEnv(raftConfig, colors),
	// }))
	// c.AddExperiment(types.NewExperiment("Random", &types.AgentConfig{
	// 	Episodes:    episodes,
	// 	Horizon:     horizon,
	// 	Policy:      types.NewRandomPolicy(),
	// 	Environment: getRaftPartEnv(raftConfig, colors),
	// }))
	// c.AddExperiment(types.NewExperiment("BonusMaxRL", &types.AgentConfig{
	// 	Episodes:    episodes,
	// 	Horizon:     horizon,
	// 	Policy:      policies.NewBonusPolicyGreedy(0.1, 0.99, 0.05),
	// 	Environment: getRaftPartEnv(raftConfig, colors),
	// }))
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
	// 	Environment: getRaftPartEnv(raftConfig, colors),
	// }))
	c.AddExperiment(types.NewExperiment(RMPolicyName, &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      RMPolicy,
		Environment: getRaftPartEnvCfg(raftConfig, colors, rlConfig),
	}))
	// c.AddExperiment(types.NewExperiment("PredHierarchy_3", &types.AgentConfig{
	// 	Episodes:    episodes,
	// 	Horizon:     horizon,
	// 	Policy:      policies.NewRewardMachinePolicy(PredHierarchy_3),
	// 	Environment: getRaftPartEnv(raftConfig, colors),
	// }))

	fmt.Print(raftConfig.String())
	fmt.Print(rlConfig.String())
	fmt.Print(PrintColors(chosenColors))

	c.Run()
}

func EtcdRaftBugsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "etcd-raft-bugs",
		Run: func(cmd *cobra.Command, args []string) {
			EtcdRaftBugs(episodes, horizon, saveFile)
		},
	}
	cmd.PersistentFlags().IntVarP(&requests, "requests", "r", 3, "Number of requests to run with")
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
