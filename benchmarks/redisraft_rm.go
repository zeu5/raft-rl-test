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
	"github.com/zeu5/raft-rl-test/util"
)

// return the list of PHs corresponding to the given command, returns empty list if unknown value
func getSetOfMachines(command string) []string {
	switch command {
	case "OnlyFollowersAndLeader":
		return []string{"OnlyFollowersAndLeader"}
	case "ElectLeader":
		return []string{"ElectLeader"}
	case "Term2":
		return []string{"Term2"}
	case "IndexAtLeast4":
		return []string{"IndexAtLeast4"}
	case "ConnectedNodes":
		return []string{"ConnectedNodes"}
	case "Bug1":
		return []string{"Bug1"}
	case "AllInSync":
		return []string{"AllInSync"}
	case "MinTerm":
		return []string{"MinTerm"}
	case "OutOfSync":
		return []string{"OutOfSync"}
	case "EntriesDifferentTerms":
		return []string{"EntriesDifferentTerms"}
	case "Set1":
		return []string{"EntriesDifferentTerms", "EntriesDifferentTermsSteps", "LogDifference3", "SyncTo6WithPReq5"}
	case "Set2":
		return []string{"EntriesDifferentTerms", "EntriesDifferentTermsSteps", "LogDifference3", "SyncTo6WithPReq5", "SyncTo7WithPReq5", "Sync7_LogDifference3", "Sync6_LogDifference4"}
	case "Set3":
		return []string{"SyncA6_PR3", "SyncA6_PR3_MT1", "SyncA6_PR3_MT1_>>_CReq1_PR2", "SA6_PR3_MT1_>>_C1T1_E1T2_PR2", "SA6_PR3_MT1_>>_2CommitsInDiffTerms"}
	case "Set4":
		return []string{"SyncA6_PR3_MT1_>>_CReq1_PR2", "SA6_PR3_MT1_>>_C1T1_E1T2_PR2", "SA6_PR3_MT1_>>_2CommitsInDiffTerms"}
	default:
		return []string{}
	}
}

type predHierState struct {
	RewFunc types.RewardFuncSingle
	Name    string
}

// adds a set of steps to a predicate hierarchy
func RedisPredHierAddBuildBlocks(pHier *policies.RewardMachine, name string) {
	var toAdd []predHierState

	switch name {

	// Sync all nodes to 6 generic entries, with 3 pending requests
	case "SyncA6_PR3":
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.NodesCommitValuesAtLeast(4).And(redisraft.PendingRequestsAtLeast(3)), Name: "SyncAll4PReq3"})
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.NodesCommitValuesAtLeast(5).And(redisraft.PendingRequestsAtLeast(3)), Name: "SyncAll5PReq3"})
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.NodesCommitValuesAtLeast(6).And(redisraft.PendingRequestsAtLeast(3)), Name: "SyncAll6PReq3"})

	// Sync all nodes to 6 generic entries, with 3 pending requests
	case "SyncA6_PR3_MT1":
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.AllNodesEntries(4, true, 1, 1, "").And(redisraft.PendingRequestsAtLeast(3)), Name: "SyncAll4PReq3MT1"})
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.AllNodesEntries(5, true, 1, 1, "").And(redisraft.PendingRequestsAtLeast(3)), Name: "SyncAll5PReq3MT1"})
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.AllNodesEntries(6, true, 1, 1, "").And(redisraft.PendingRequestsAtLeast(3)), Name: "SyncAll6PReq3MT1"})

	// Sync all nodes to 6 generic entries, with 3 pending requests
	case "SyncA6_PR3_MT1_>>_CReq1_PR2":
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.AllNodesEntries(4, true, 1, 1, "").And(redisraft.PendingRequestsAtLeast(3)), Name: "SyncAll4PReq3MT1"})
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.AllNodesEntries(5, true, 1, 1, "").And(redisraft.PendingRequestsAtLeast(3)), Name: "SyncAll5PReq3MT1"})
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.AllNodesEntries(6, true, 1, 1, "").And(redisraft.PendingRequestsAtLeast(3)), Name: "SyncAll6PReq3MT1"})
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.AllNodesEntries(6, true, 1, 1, "").And(redisraft.PendingRequestsAtLeast(2)).And(redisraft.AllNodesEntries(1, true, 1, 1, "NORMAL")), Name: "Comm1PReq2MT1"})

	case "InitSync_C1T1_EntryT2_PReq2":
		RedisPredHierAddBuildBlocks(pHier, "SyncA6_PR3_MT1_>>_CReq1_PR2")
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.AllNodesEntries(1, true, 2, 2, "NORMAL").And(redisraft.PendingRequestsAtLeast(2)), Name: "Entry2PReq2"})

	}

	for _, state := range toAdd {
		pHier.AddState(state.RewFunc, state.Name)
	}

}

func getRedisPredicateHeirarchy(name string) (*policies.RewardMachine, bool, bool) {
	var machine *policies.RewardMachine = nil
	oneTime := false
	switch name {
	case "OnlyFollowersAndLeader":
		// This is always true initially
		machine = policies.NewRewardMachine(redisraft.OnlyFollowersAndLeader())
		oneTime = true
	case "ElectLeader":
		// This is also always true. The system starts with a config where the leader is elected
		machine = policies.NewRewardMachine(redisraft.LeaderElected())
		oneTime = true
	case "Term2":
		machine = policies.NewRewardMachine(redisraft.TermNumber(2))
	case "IndexAtLeast4":
		machine = policies.NewRewardMachine(redisraft.CurrentIndexAtLeast(4))
	case "ConnectedNodes":
		// The leader node will always have connected nodes. Also always true
		machine = policies.NewRewardMachine(redisraft.NumConnectedNodesInAny(3))
	case "Bug1":
		machine = policies.NewRewardMachine(redisraft.CurrentIndexAtLeast(5).And(redisraft.NumConnectedNodesInAny(3)))
		machine.AddState(redisraft.OnlyFollowersAndLeader(), "OnlyFollowersAndLeader")
		machine.AddState(redisraft.CurrentIndexAtLeast(5), "Atleast9Entries")
		machine.AddState(redisraft.InState("follower").And(redisraft.CurrentIndexAtLeast(5)), "FollowerAtLeastIndex6")
	case "AllInSync":
		machine = policies.NewRewardMachine(redisraft.AllInSyncAtleast(2))
		machine.AddState(redisraft.TermNumber(2), "ReachTerm2")
		machine.AddState(redisraft.AllInTermAtleast(2), "AllReachTerm2")
	case "MinTerm":
		machine = policies.NewRewardMachine(redisraft.AllInTermAtleast(2))
		machine.AddState(redisraft.AllInTermAtleast(1), "MinTerm1")
		machine.AddState(redisraft.InState("candidate").And(redisraft.AllInTermAtleast(1)), "Candidate")
	case "OutOfSync":
		machine = policies.NewRewardMachine(redisraft.AllInSyncAtleast(3))
		machine.AddState(redisraft.OutOfSyncBy(2), "OutOfSync")
	case "EntriesDifferentTerms":
		machine = policies.NewRewardMachine(redisraft.EntriesInDifferentTerms())
	case "EntriesDifferentTermsSteps":
		machine = policies.NewRewardMachine(redisraft.EntriesInDifferentTerms())
		machine.AddState(redisraft.NodesCommitValuesAtLeast(4).And(redisraft.PendingRequestsAtLeast(5)), "SyncAll4PReq5")
		machine.AddState(redisraft.NodesCommitValuesAtLeast(6).And(redisraft.PendingRequestsAtLeast(5)), "SyncAll6PReq5")
		machine.AddState(redisraft.NodesCommitValuesAtLeast(6).And(redisraft.PendingRequestsAtLeast(4)).And(redisraft.AtLeastOneCommittedNormalEntry()), "SyncAll6PReq4Commit1")
	case "LogDifference3":
		machine = policies.NewRewardMachine(redisraft.NodesCommitValuesAtLeast(4).And(redisraft.DiffInCommittedEntries(3)))
		machine.AddState(redisraft.NodesCommitValuesAtLeast(4).And(redisraft.PendingRequestsAtLeast(5)).And(redisraft.OnlyFollowersAndLeaderInTerm(1)), "SyncAll4PReq5")
		machine.AddState(redisraft.NodesCommitValuesAtLeast(4).And(redisraft.DiffInCommittedEntries(1)), "Diff1")
		machine.AddState(redisraft.NodesCommitValuesAtLeast(4).And(redisraft.DiffInCommittedEntries(2)), "Diff2")
		oneTime = true
	case "SyncTo6WithPReq5":
		machine = policies.NewRewardMachine(redisraft.NodesCommitValuesAtLeast(6).And(redisraft.PendingRequestsAtLeast(5)))
		machine.AddState(redisraft.NodesCommitValuesAtLeast(4).And(redisraft.PendingRequestsAtLeast(5)), "SyncAll4PReq5")
		machine.AddState(redisraft.NodesCommitValuesAtLeast(5).And(redisraft.PendingRequestsAtLeast(5)), "SyncAll5PReq5")
		oneTime = true
	case "SyncTo7WithPReq5":
		machine = policies.NewRewardMachine(redisraft.NodesCommitValuesAtLeast(7).And(redisraft.PendingRequestsAtLeast(5)))
		machine.AddState(redisraft.NodesCommitValuesAtLeast(4).And(redisraft.PendingRequestsAtLeast(5)), "SyncAll4PReq5")
		machine.AddState(redisraft.NodesCommitValuesAtLeast(5).And(redisraft.PendingRequestsAtLeast(5)), "SyncAll5PReq5")
		machine.AddState(redisraft.NodesCommitValuesAtLeast(6).And(redisraft.PendingRequestsAtLeast(5)), "SyncAll6PReq5")
		oneTime = true
	case "Sync7_LogDifference4":
		machine = policies.NewRewardMachine(redisraft.NodesCommitValuesAtLeast(7).And(redisraft.DiffInCommittedEntries(4)))
		machine.AddState(redisraft.NodesCommitValuesAtLeast(4).And(redisraft.PendingRequestsAtLeast(5)), "SyncAll4PReq5")
		machine.AddState(redisraft.NodesCommitValuesAtLeast(5).And(redisraft.PendingRequestsAtLeast(5)), "SyncAll5PReq5")
		machine.AddState(redisraft.NodesCommitValuesAtLeast(6).And(redisraft.PendingRequestsAtLeast(5)), "SyncAll6PReq5")
		machine.AddState(redisraft.NodesCommitValuesAtLeast(7).And(redisraft.PendingRequestsAtLeast(5)), "SyncAll7PReq5")
		machine.AddState(redisraft.NodesCommitValuesAtLeast(7).And(redisraft.DiffInCommittedEntries(1)).And(redisraft.PendingRequestsAtLeast(4)), "Diff1_PReq4")
		machine.AddState(redisraft.NodesCommitValuesAtLeast(7).And(redisraft.DiffInCommittedEntries(2)).And(redisraft.PendingRequestsAtLeast(3)), "Diff2_PReq3")
		machine.AddState(redisraft.NodesCommitValuesAtLeast(7).And(redisraft.DiffInCommittedEntries(3)).And(redisraft.PendingRequestsAtLeast(2)), "Diff3_PReq2")
		oneTime = true
	case "Sync7_LogDifference3":
		machine = policies.NewRewardMachine(redisraft.NodesCommitValuesAtLeast(7).And(redisraft.DiffInCommittedEntries(3)))
		machine.AddState(redisraft.NodesCommitValuesAtLeast(4).And(redisraft.PendingRequestsAtLeast(5)), "SyncAll4PReq5")
		machine.AddState(redisraft.NodesCommitValuesAtLeast(5).And(redisraft.PendingRequestsAtLeast(5)), "SyncAll5PReq5")
		machine.AddState(redisraft.NodesCommitValuesAtLeast(6).And(redisraft.PendingRequestsAtLeast(5)), "SyncAll6PReq5")
		machine.AddState(redisraft.NodesCommitValuesAtLeast(7).And(redisraft.PendingRequestsAtLeast(5)), "SyncAll7PReq5")
		machine.AddState(redisraft.NodesCommitValuesAtLeast(7).And(redisraft.DiffInCommittedEntries(1)).And(redisraft.PendingRequestsAtLeast(4)), "Diff1_PReq4")
		machine.AddState(redisraft.NodesCommitValuesAtLeast(7).And(redisraft.DiffInCommittedEntries(2)).And(redisraft.PendingRequestsAtLeast(3)), "Diff2_PReq3")
		oneTime = true
	case "Sync6_LogDifference4":
		machine = policies.NewRewardMachine(redisraft.NodesCommitValuesAtLeast(6).And(redisraft.DiffInCommittedEntries(4)))
		machine.AddState(redisraft.NodesCommitValuesAtLeast(4).And(redisraft.PendingRequestsAtLeast(5)), "SyncAll4PReq5")
		machine.AddState(redisraft.NodesCommitValuesAtLeast(5).And(redisraft.PendingRequestsAtLeast(5)), "SyncAll5PReq5")
		machine.AddState(redisraft.NodesCommitValuesAtLeast(6).And(redisraft.PendingRequestsAtLeast(5)), "SyncAll6PReq5")
		machine.AddState(redisraft.NodesCommitValuesAtLeast(6).And(redisraft.DiffInCommittedEntries(1)).And(redisraft.PendingRequestsAtLeast(4)), "Diff1_PReq4")
		machine.AddState(redisraft.NodesCommitValuesAtLeast(6).And(redisraft.DiffInCommittedEntries(2)).And(redisraft.PendingRequestsAtLeast(3)), "Diff2_PReq3")
		machine.AddState(redisraft.NodesCommitValuesAtLeast(6).And(redisraft.DiffInCommittedEntries(3)).And(redisraft.PendingRequestsAtLeast(2)), "Diff3_PReq2")
		oneTime = true

	case "SyncA6_PR3":
		machine = policies.NewRewardMachine(redisraft.NodesCommitValuesAtLeast(6).And(redisraft.PendingRequestsAtLeast(3)))
		machine.AddState(redisraft.NodesCommitValuesAtLeast(4).And(redisraft.PendingRequestsAtLeast(3)), "SyncAll4PReq3")
		machine.AddState(redisraft.NodesCommitValuesAtLeast(5).And(redisraft.PendingRequestsAtLeast(3)), "SyncAll5PReq3")
		oneTime = true

	case "SyncA6_PR3_MT1":
		machine = policies.NewRewardMachine(redisraft.NodesCommitValuesAtLeastMaxTerm(6, 1).And(redisraft.PendingRequestsAtLeast(3)))
		machine.AddState(redisraft.AllNodesEntries(4, true, 1, 1, "").And(redisraft.PendingRequestsAtLeast(3)), "SyncAll4PReq3Mt1")
		machine.AddState(redisraft.AllNodesEntries(5, true, 1, 1, "").And(redisraft.PendingRequestsAtLeast(3)), "SyncAll5PReq3Mt1")
		oneTime = true

	case "SyncA6_PR3_MT1_>>_CReq1_PR2":
		machine = policies.NewRewardMachine(redisraft.AllNodesEntries(6, true, 1, 1, "").And(redisraft.PendingRequestsAtLeast(2)).And(redisraft.AllNodesEntries(1, true, 1, 1, "NORMAL")))
		RedisPredHierAddBuildBlocks(machine, "SyncA6_PR3_MT1")
		oneTime = true

	case "SA6_PR3_MT1_>>_C1T1_E1T2_PR2":
		machine = policies.NewRewardMachine(redisraft.AllNodesEntries(6, true, 1, 1, "").And(redisraft.PendingRequestsAtLeast(2)).And(redisraft.AllNodesEntries(1, true, 1, 1, "NORMAL")).And(redisraft.AtLeastOneNodeEntries(1, false, 2, 2, "")))
		RedisPredHierAddBuildBlocks(machine, "SyncA6_PR3_MT1_>>_CReq1_PR2")
		oneTime = true

	case "SA6_PR3_MT1_>>_2CommitsInDiffTerms":
		machine = policies.NewRewardMachine(redisraft.AllNodesEntries(6, true, 1, 1, "").And(redisraft.AllNodesEntries(1, true, 1, 1, "NORMAL")).And(redisraft.AllNodesEntries(1, true, 2, 10, "NORMAL")))
		RedisPredHierAddBuildBlocks(machine, "SyncA6_PR3_MT1_>>_CReq1_PR2")
		oneTime = true
	}

	return machine, oneTime, machine != nil
}

func RedisRaftRM(machine string, episodes, horizon int, saveFile string, ctx context.Context) {
	clusterConfig := redisraft.ClusterConfig{
		NumNodes:            3,
		BasePort:            5000,
		BaseInterceptPort:   2023,
		ID:                  1,
		InterceptListenAddr: "localhost:7074",
		WorkingDir:          path.Join(saveFile, "tmp"),
		NumRequests:         5,

		RequestTimeout:  25,  // heartbeat in milliseconds (fixed or variable?)
		ElectionTimeout: 150, // election timeout in milliseconds (from specified value to its double)

		TickLength: 20,
	}
	env := redisraft.NewRedisRaftEnv(ctx, &clusterConfig, path.Join(saveFile, "tickLength"))
	// env.SetPrintStats(true) // to print the episode stats
	defer env.Cleanup()

	// abstraction for both plot and RL
	availableColors := make(map[string]redisraft.RedisRaftColorFunc)
	availableColors["state"] = redisraft.ColorState()   // replica internal state
	availableColors["commit"] = redisraft.ColorCommit() // number of committed entries? includes config changes, leader elect, request entry
	availableColors["leader"] = redisraft.ColorLeader() // if a replica is leader? boolean?
	availableColors["vote"] = redisraft.ColorVote()     // ?
	availableColors["index"] = redisraft.ColorIndex()   // next available index to write?
	availableColors["boundedTerm3"] = redisraft.ColorBoundedTerm(3)
	availableColors["boundedTerm5"] = redisraft.ColorBoundedTerm(5)   // current term, bounded to the passed value
	availableColors["boundedTerm10"] = redisraft.ColorBoundedTerm(10) // current term, bounded to the passed value
	availableColors["applied"] = redisraft.ColorApplied()
	availableColors["snapshot"] = redisraft.ColorSnapshot()
	availableColors["log"] = redisraft.ColorLog()
	availableColors["boundedLog3"] = redisraft.ColorBoundedLog(3)
	availableColors["boundedLog5"] = redisraft.ColorBoundedLog(5)
	availableColors["boundedLog10"] = redisraft.ColorBoundedLog(10)

	chosenColors := []string{
		"state",
		"commit",
		"leader",
		"vote",
		"boundedTerm5",
		"index",
		// "snapshot",
		// "log",
		"boundedLog5",
	}

	colors := make([]redisraft.RedisRaftColorFunc, 0)
	for _, color := range chosenColors {
		colors = append(colors, availableColors[color])
	}

	// colors := []redisraft.RedisRaftColorFunc{redisraft.ColorState(), redisraft.ColorCommit(), redisraft.ColorLeader(), redisraft.ColorVote(), redisraft.ColorBoundedTerm(5), redisraft.ColorIndex(), redisraft.ColorSnapshot()}

	partitionEnvConfig := types.PartitionEnvConfig{
		Painter:                redisraft.NewRedisRaftStatePainter(colors...),
		Env:                    env,
		TicketBetweenPartition: 5,
		MaxMessagesPerTick:     100,
		StaySameStateUpto:      5,
		NumReplicas:            3,
		WithCrashes:            false,
		CrashLimit:             3,
		MaxInactive:            1,
	}

	c := types.NewComparison(&types.ComparisonConfig{
		Runs:       runs,
		Episodes:   episodes,
		Horizon:    horizon,
		RecordPath: saveFile,
		Timeout:    10 * time.Second,
		// record flags
		RecordTraces: false,
		RecordTimes:  true,
		RecordPolicy: false,
		// last traces
		PrintLastTraces:     20,
		PrintLastTracesFunc: redisraft.ReadableTracePrintable,
		// report config
		ReportConfig: types.RepConfigStandard(),
	})

	c.AddAnalysis("Plot", redisraft.NewCoverageAnalyzer(colors...), redisraft.CoverageComparator(saveFile))
	c.AddAnalysis("Crashes", redisraft.NewBugCrashAnalyzer(path.Join(saveFile, "crash")), redisraft.BugComparator())
	c.AddAnalysis("Bugs", redisraft.NewBugAnalyzer(
		path.Join(saveFile, "bugs"),
		types.BugDesc{Name: "ReducedLog", Check: redisraft.ReducedLog()},
		types.BugDesc{Name: "ModifiedLog", Check: redisraft.ModifiedLog()},
		types.BugDesc{Name: "InconsistentLogs", Check: redisraft.InconsistentLogs()},
		// types.BugDesc{Name: "True", Check: redisraft.TruePredicate()},
		// types.BugDesc{Name: "DifferentTermsEntries", Check: redisraft.EntriesInDifferentTermsDummy()},
	), types.BugComparator(path.Join(saveFile, "bugs")))

	machines := getSetOfMachines(machine)
	pHierarchiesPolicies := make(map[string]*policies.RewardMachinePolicy) // map of PH policies
	for _, pHierName := range machines {                                   // for each of them create it and create an analyzer
		rm, oneTime, ok := getRedisPredicateHeirarchy(pHierName)
		if !ok { // if something goes wrong just skip it
			continue
		}
		RMPolicy := policies.NewRewardMachinePolicy(rm, oneTime)
		pHierarchiesPolicies[pHierName] = RMPolicy

		// c.AddAnalysis("plot", redisraft.CoverageAnalyzer(colors...), redisraft.CoverageComparator(saveFile))
		c.AddAnalysis(pHierName, policies.NewRewardMachineAnalyzer(RMPolicy), policies.RewardMachineCoverageComparator(saveFile, pHierName))
	}

	for pHierName, policy := range pHierarchiesPolicies {
		c.AddExperiment(types.NewExperiment(pHierName, policy, types.NewPartitionEnv(partitionEnvConfig)))
	}

	c.AddExperiment(types.NewExperiment(
		"rl",
		policies.NewBonusPolicyGreedy(0.1, 0.99, 0.05),
		types.NewPartitionEnv(partitionEnvConfig),
	))
	c.AddExperiment(types.NewExperiment(
		"random",
		types.NewRandomPolicy(),
		types.NewPartitionEnv(partitionEnvConfig),
	))

	// strict := policies.NewStrictPolicy(types.NewRandomPolicy())
	// strict.AddPolicy(policies.If(policies.Always()).Then(types.PickKeepSame()))

	// c.AddExperiment(types.NewExperiment("Strict", &types.AgentConfig{
	// 	Episodes:    episodes,
	// 	Horizon:     horizon,
	// 	Policy:      strict,
	// 	Environment: types.NewPartitionEnv(partitionEnvConfig),
	// }))

	// print config file
	configPath := path.Join(saveFile, "config.txt")
	util.WriteToFile(configPath, ExperimentParametersPrintable(), clusterConfig.Printable(), partitionEnvConfig.Printable(), PrintColors(chosenColors))

	c.Run(ctx)
}

func RedisRaftRMCommand() *cobra.Command {
	return &cobra.Command{
		Use:  "redisraft-rm [machine]",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt) // channel for interrupts from os

			doneCh := make(chan struct{}) // channel for done signal from application

			ctx, cancel := context.WithCancel(context.Background())
			go func() { // start a go-routine
				select { // can wait on multiple channels
				case <-sigCh:
				case <-doneCh:
				}
				cancel()
			}()

			RedisRaftRM(args[0], episodes, horizon, saveFile, ctx)

			close(doneCh)
		},
	}
}
