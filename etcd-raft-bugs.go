package main

import (
	"path"

	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/raft"
	"github.com/zeu5/raft-rl-test/types"
)

func EtcdRaftBugs(episodes, horizon int, savePath string) {
	// config of the running system
	requests = 3
	raftConfig := raft.RaftEnvironmentConfig{
		Replicas:      3,
		ElectionTick:  6, // lower bound for a process to try to go to new term (starting an election) - double of this is upperbound
		HeartbeatTick: 3, // frequency of heartbeats
		Timeouts:      timeouts,
		Requests:      requests,
	}

	// abstraction for both plot and RL
	// colors is one abstraction definition
	colors := []raft.RaftColorFunc{
		raft.ColorState(),        // replica internal state
		raft.ColorCommit(),       // number of committed entries? includes config changes, leader elect, request entry
		raft.ColorLeader(),       // if a replica is leader? boolean?
		raft.ColorVote(),         // ?
		raft.ColorBoundedTerm(5), // current term, bounded to the passed value
	}

	// build a machine with sequence of predicates
	// elect a leader, commit with one replica not updated, elect that replica as leader
	PredHierarchy_1 := policies.NewRewardMachine(raft.LeaderElectedPredicateSpecific(3).And(raft.EmptyLogSpecific(3)).And(raft.AtLeastOneLogNotEmpty())) // final predicate - target space
	PredHierarchy_1.AddState(raft.LeaderElectedPredicateSpecific(1).Or(raft.LeaderElectedPredicateSpecific(2)), "LeaderElected(1)Or(2)")                 // 1st step
	PredHierarchy_1.AddState(raft.EmptyLogSpecific(3).And(raft.AtLeastOneLogNotEmpty()), "CommitWithEmptyLog(3)")                                        // 2nd ...

	// elect a leader, commit a request and then elect another leader
	PredHierarchy_2 := policies.NewRewardMachine(raft.EntriesInDifferentTermsInLog(2))
	PredHierarchy_2.AddState(raft.LeaderElectedPredicateState(), "LeaderElected")
	PredHierarchy_2.AddState(raft.AtLeastOneLogOneEntry(), "OneEntryLog")
	PredHierarchy_2.AddState(raft.AtLeastOneLogOneEntryPlusReplicaInHigherTerm(), "OneEntryLogAndReplicaInHigherTerm")
	PredHierarchy_2.AddState(raft.AtLeastOneLogOneEntryPlusSubsequentLeaderElection(), "OneEntryLogAndSubsequentLeaderElection")

	// elect a leader and commit a request
	PredHierarchy_3 := policies.NewRewardMachine(raft.EntriesInDifferentTermsInLog(2))
	PredHierarchy_3.AddState(raft.LeaderElectedPredicateState(), "LeaderElected")
	PredHierarchy_3.AddState(raft.AtLeastOneLogNotEmpty().And(raft.StackSizeLowerBound(2)), "EntryCommitted And TwoAvailableRequests")
	PredHierarchy_3.AddState(raft.AtLeastOneEntryANDSubsequentLeaderElection().And(raft.StackSizeLowerBound(2)), "OneEntryLogAndSubsequentLeaderElection And TwoAvailableRequests")

	// elect a leader and commit a request
	PredHierarchy_4 := policies.NewRewardMachine(raft.EntriesInDifferentTermsInLog(2))
	PredHierarchy_4.AddState(raft.LeaderElectedPredicateSpecific(1), "LeaderElected(1)")
	PredHierarchy_4.AddState(raft.LeaderElectedPredicateSpecific(1).And(raft.ExactEntriesInLogSpecific(1, 1)), "CommitOneEntry(1)")
	PredHierarchy_4.AddState(raft.LeaderElectedPredicateSpecific(2).And(raft.ExactEntriesInLogSpecific(1, 1)), "ChangeOfLeader(1->2)")

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
		types.BugDesc{Name: "DummyBug", Check: raft.DummyBug()},
	), types.BugComparator(saveFile))

	// c.AddAnalysis("CommitOnlyOneEntry", policies.RewardMachineAnalyzer(PredHierarchy_3), policies.RewardMachineCoverageComparator(saveFile))
	c.AddAnalysis("CommitEntriesInDifferentTerms", policies.RewardMachineAnalyzer(PredHierarchy_3), policies.RewardMachineCoverageComparator(saveFile))

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
	// c.AddExperiment(types.NewExperiment("PredHierarchy_4", &types.AgentConfig{
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
	c.AddExperiment(types.NewExperiment("PredHierarchy_3", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      policies.NewRewardMachinePolicy(PredHierarchy_3),
		Environment: getRaftPartEnv(raftConfig, colors),
	}))
	// c.AddExperiment(types.NewExperiment("PredHierarchy_3", &types.AgentConfig{
	// 	Episodes:    episodes,
	// 	Horizon:     horizon,
	// 	Policy:      policies.NewRewardMachinePolicy(PredHierarchy_3),
	// 	Environment: getRaftPartEnv(raftConfig, colors),
	// }))

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
