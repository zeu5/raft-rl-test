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
	raftConfig := raft.RaftEnvironmentConfig{
		Replicas:      3,
		ElectionTick:  10, // lower bound for a process to try to go to new term (starting an election) - double of this is upperbound
		HeartbeatTick: 3,  // frequency of heartbeats
		Timeouts:      timeouts,
		Requests:      requests,
	}

	// abstraction for both plot and RL
	// colors is one abstraction definition
	colors := []raft.RaftColorFunc{raft.ColorState(), raft.ColorCommit(), raft.ColorLeader(), raft.ColorVote(), raft.ColorBoundedTerm(5)}

	// build a machine with sequence of predicates
	// elect a leader, commit with one replica not updated, elect that replica as leader
	PredHierarchy_1 := policies.NewRewardMachine(raft.LeaderElectedPredicateSpecific(3).And(raft.EmptyLogSpecific(3)).And(raft.AtLeastOneLogNotEmpty())) // final predicate - target space
	PredHierarchy_1.AddState(raft.LeaderElectedPredicateSpecific(1).Or(raft.LeaderElectedPredicateSpecific(2)), "LeaderElected(1)Or(2)")                 // 1st step
	PredHierarchy_1.AddState(raft.EmptyLogSpecific(3).And(raft.AtLeastOneLogNotEmpty()), "CommitWithEmptyLog(3)")                                        // 2nd ...

	// elect a leader, commit a request and then elect another leader
	PredHierarchy_2 := policies.NewRewardMachine((raft.LeaderElectedPredicateSpecific(3).Or(raft.LeaderElectedPredicateSpecific(2))).And(raft.AtLeastOneLogNotEmpty()))
	PredHierarchy_2.AddState(raft.LeaderElectedPredicateSpecific(1), "LeaderElected(1)")
	PredHierarchy_2.AddState(raft.LeaderElectedPredicateSpecific(1).And(raft.AtLeastOneLogNotEmpty()), "CommitWithLeaderElected(1)")

	// elect a leader and commit a request
	PredHierarchy_3 := policies.NewRewardMachine(raft.LeaderElectedPredicate().And(raft.AtLeastOneLogNotEmpty()))
	PredHierarchy_3.AddState(raft.LeaderElectedPredicate(), "LeaderElected")

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
	), types.BugComparator(saveFile))

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
	c.AddExperiment(types.NewExperiment("BonusMaxRL", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      policies.NewBonusPolicyGreedy(0.1, 0.99, 0.2),
		Environment: getRaftPartEnv(raftConfig, colors),
	}))
	c.AddExperiment(types.NewExperiment("PredHierarchy_1", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      policies.NewRewardMachinePolicy(PredHierarchy_1),
		Environment: getRaftPartEnv(raftConfig, colors),
	}))
	c.AddExperiment(types.NewExperiment("PredHierarchy_2", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      policies.NewRewardMachinePolicy(PredHierarchy_2),
		Environment: getRaftPartEnv(raftConfig, colors),
	}))
	c.AddExperiment(types.NewExperiment("PredHierarchy_3", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      policies.NewRewardMachinePolicy(PredHierarchy_3),
		Environment: getRaftPartEnv(raftConfig, colors),
	}))

	c.Run()
}

func EtcdRaftBugsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "etcd-raft-bugs",
		Run: func(cmd *cobra.Command, args []string) {
			EtcdRaftBugs(episodes, horizon, saveFile)
		},
	}
	cmd.PersistentFlags().IntVarP(&requests, "requests", "r", 2, "Number of requests to run with")
	return cmd
}
