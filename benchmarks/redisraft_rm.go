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
	case "debug":
		return []string{"OneTerm2"}
	case "expl":
		return []string{"neg", "baselines"}
	case "set1":
		return []string{"OneTerm2", "AllInTerm2", "LeaderInTerm2[1]", "LeaderInTerm2[2]", "LeaderInTerm2[3]",
			"LogDiff1[1]", "LogCommitDiff3[1]", "LogCommitDiff3[2]", "LogCommitDiff3[4]",
			"baselines"}
	case "set2":
		return []string{"EntryInTerm2[1]", "EntryInTerm2[3]",
			"MoreThanOneCandidate", "MoreThanOneLeader",
			"InconsistentLogs[1]", "InconsistentLogs[2]",
			"baselines"}
	case "total":
		return []string{"LeaderInTerm2[1]", "LeaderInTerm2[2]",
			"LogDiff1[1]", "LogDiff1[1]WithLeader",
			"LogCommitDiff3[1]", "LogCommitDiff3[2]", "LogCommitDiff3[4]",
			"EntryInTerm2[1]", "EntryInTerm2[3]",
			"MoreThanOneCandidate", "MoreThanOneLeader",
			"InconsistentLogs[1]", "InconsistentLogs[2]",
			"baselines"}
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
	case "SyncA5_PR2_MT1":
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.AllNodesEntries(3, true, 1, 1, "").And(redisraft.PendingRequestsAtLeast(2)), Name: "SyncAll3PReq2MT1"})
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.AllNodesEntries(4, true, 1, 1, "").And(redisraft.PendingRequestsAtLeast(2)), Name: "SyncAll4PReq2MT1"})
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.AllNodesEntries(5, true, 1, 1, "").And(redisraft.PendingRequestsAtLeast(2)), Name: "SyncAll5PReq2MT1"})

	// Sync all nodes to 6 generic entries, with 3 pending requests
	case "SyncA5_MT1":
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.AllNodesEntries(3, true, 1, 1, ""), Name: "SyncAll3MT1"})
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.AllNodesEntries(4, true, 1, 1, ""), Name: "SyncAll4MT1"})
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.AllNodesEntries(5, true, 1, 1, ""), Name: "SyncAll5MT1"})

	// Sync all nodes to 6 generic entries, with 3 pending requests
	case "SyncA5_MT1_NoNormal":
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.AllNodesEntries(3, true, 1, 1, "").And((redisraft.AtLeastOneNodeEntries(1, false, 1, 1, "NORMAL").Not())), Name: "SyncAll3MT1"})
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.AllNodesEntries(4, true, 1, 1, "").And((redisraft.AtLeastOneNodeEntries(1, false, 1, 1, "NORMAL").Not())), Name: "SyncAll4MT1"})
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.AllNodesEntries(5, true, 1, 1, "").And((redisraft.AtLeastOneNodeEntries(1, false, 1, 1, "NORMAL").Not())), Name: "SyncAll5MT1"})
	}

	for _, state := range toAdd {
		pHier.AddState(state.RewFunc, state.Name)
	}

}

func getRedisPredicateHeirarchy(name string) (*policies.RewardMachine, bool, bool) {
	var machine *policies.RewardMachine = nil
	oneTime := false
	switch name {
	case "2CommT1X":
		// Requires a commit in term 1 (NORMAL entry, apparently it is only achievable by sending a request), and another commit in a term > 1.
		// It includes several intermediate steps: request commit in term1, processes in higher terms, all processes in a single term > 1, a leader elected in a term > 1.
		// (after initial sync to 5 entries)
		machine = policies.NewRewardMachine(redisraft.AllNodesEntries(6, true, 1, 1, "").
			And(redisraft.AllNodesEntries(1, true, 1, 1, "NORMAL")).
			And(redisraft.AllNodesTerms(2, 5)).
			// And(redisraft.AllNodesEntries(2, true, 2, 2, "")).
			And(redisraft.AllNodesEntries(1, true, 2, 5, "")))
		RedisPredHierAddBuildBlocks(machine, "Sync_C1T1_LeaderX")
		oneTime = true

	case "2Comm-T(12)-T(35)":
		machine = policies.NewRewardMachine(redisraft.AllNodesEntries(5, true, 1, 1, "").
			And(redisraft.AllNodesEntries(1, true, 1, 2, "NORMAL")).
			And(redisraft.AllNodesEntries(1, true, 3, 5, "NORMAL")))
		oneTime = true

	case "LogDiff3-PreSync":
		// Reach a difference of at least 3 entries in the length of two processes' committed logs.
		// (after initial sync to 5 entries)
		machine = policies.NewRewardMachine(redisraft.AllNodesEntries(5, true, 1, 1, "").
			And(redisraft.DiffInCommittedEntries(3)))
		RedisPredHierAddBuildBlocks(machine, "SyncA5_MT1_NoNormal")
		oneTime = true

	case "LogDiff4":
		machine = policies.NewRewardMachine(redisraft.AllNodesEntries(5, true, 1, 1, "").
			And(redisraft.DiffInCommittedEntries(4)))
		RedisPredHierAddBuildBlocks(machine, "SyncA5_MT1_NoNormal")
		oneTime = true

	case "LogDiff3_Steps":
		// Reach a difference of at least 3 entries in the length of two processes' committed logs.
		// It includes intermediate steps: difference 1, difference 2, difference 3 in the logs (does not require committed entries)
		// (after initial sync to 5 entries)
		machine = policies.NewRewardMachine(redisraft.AllNodesEntries(5, true, 1, 1, "").
			And(redisraft.DiffInCommittedEntries(3)))
		RedisPredHierAddBuildBlocks(machine, "SyncA5_MT1_NoNormal")
		machine.AddState(redisraft.AllNodesEntries(5, true, 1, 1, "").And(redisraft.DiffInEntries(1)), "Diff1")
		machine.AddState(redisraft.AllNodesEntries(5, true, 1, 1, "").And(redisraft.DiffInEntries(2)), "Diff2")
		machine.AddState(redisraft.AllNodesEntries(5, true, 1, 1, "").And(redisraft.DiffInEntries(3)), "Diff3")
		oneTime = true

	case "CommT1_LeadT25_LogDiff3":
		// Reach a difference of at least 3 entries in the length of two processes' committed logs after a commit in Term 1 and a leader election in a term > 1.
		// It includes several intermediate steps
		// (after initial sync to 5 entries)
		machine = policies.NewRewardMachine(redisraft.AllNodesEntries(6, true, 1, 1, "").
			And(redisraft.AllNodesEntries(1, true, 1, 1, "NORMAL")).
			And(redisraft.AllNodesTerms(2, 5)).
			// And(redisraft.AllNodesEntries(2, true, 2, 2, "")).
			And(redisraft.AtLeastOneNodeStates([]string{"leader"})).
			And(redisraft.DiffInCommittedEntries(3)))
		RedisPredHierAddBuildBlocks(machine, "Sync_C1T1_LeaderX")
		Commit1T1 := redisraft.AllNodesEntries(6, true, 1, 1, "").And(redisraft.AllNodesEntries(1, true, 1, 1, "NORMAL")).And((redisraft.AtLeastOneNodeEntries(3, false, 1, 1, "NORMAL").Not()))
		Sync_C1T1_LeaderX := redisraft.AtLeastOneNodeStates([]string{"leader"}).And(redisraft.AllNodesTerms(2, 5).And(Commit1T1))
		machine.AddState(Sync_C1T1_LeaderX.And(redisraft.DiffInEntries(1)), "C1T1_AllInT2X_Leader_Diff1")
		machine.AddState(Sync_C1T1_LeaderX.And(redisraft.DiffInEntries(2)), "C1T1_AllInT2X_Leader_Diff2")
		oneTime = true

	case "OneTerm2":
		// At least one node in term 2
		machine = policies.NewRewardMachine(redisraft.AtLeastOneNodeTerm(2, 2))
		oneTime = true

	case "AllInTerm2":
		// all nodes in term 2
		machine = policies.NewRewardMachine(redisraft.AllNodesTerms(2, 2))
		oneTime = true

	case "LeaderInTerm2[1]":
		// all nodes in term 2 and one leader elected
		machine = policies.NewRewardMachine(redisraft.AtLeastOneNodeStatesInTerm([]string{"leader"}, 2, 2))
		oneTime = true

	case "LeaderInTerm2[2]":
		// all nodes in term 2 and one leader elected
		machine = policies.NewRewardMachine(redisraft.AtLeastOneNodeStatesInTerm([]string{"leader"}, 2, 2))
		machine.AddState(redisraft.AtLeastOneNodeTerm(2, 2), "OneInTerm2")
		oneTime = true

	case "LogCommitDiff3[1]":
		// Reach a difference of at least 3 entries in the length of two processes' committed logs.
		// (after initial sync to 5 entries)
		machine = policies.NewRewardMachine(redisraft.AllNodesEntries(5, true, 1, 1, "").
			And(redisraft.DiffInCommittedEntries(3)))
		oneTime = true

	case "LogCommitDiff3[2]":
		// Reach a difference of at least 3 entries in the length of two processes' committed logs.
		// (after initial sync to 5 entries)
		machine = policies.NewRewardMachine(redisraft.AllNodesEntries(5, true, 1, 1, "").
			And(redisraft.DiffInCommittedEntries(3)))
		machine.AddState(redisraft.AllNodesEntries(5, true, 1, 1, "").And(redisraft.DiffInEntries(1)), "LogDiff1")
		oneTime = true

	case "LogCommitDiff3[4]":
		// Reach a difference of at least 3 entries in the length of two processes' committed logs.
		// (after initial sync to 5 entries)
		machine = policies.NewRewardMachine(redisraft.AllNodesEntries(5, true, 1, 1, "").
			And(redisraft.DiffInCommittedEntries(3)))
		machine.AddState(redisraft.AllNodesEntries(5, true, 1, 1, "").And(redisraft.DiffInEntries(1)), "LogDiff1")
		machine.AddState(redisraft.AllNodesEntries(5, true, 1, 1, "").And(redisraft.DiffInEntries(2)), "LogDiff2")
		machine.AddState(redisraft.AllNodesEntries(5, true, 1, 1, "").And(redisraft.DiffInEntries(3)), "LogDiff3")
		oneTime = true

	case "LogDiff1[1]":
		machine = policies.NewRewardMachine(redisraft.AllNodesEntries(5, true, 1, 1, "").And(redisraft.DiffInEntries(1)))
		oneTime = true

	case "LogDiff1[1]WithLeader":
		machine = policies.NewRewardMachine(redisraft.AllNodesEntries(5, true, 1, 1, "").And(redisraft.DiffInEntries(1)).And(redisraft.AtLeastOneNodeStates([]string{"leader"})))
		oneTime = true

	// case "NEntriesInAny":
	// 	machine = policies.NewRewardMachine(redisraft.AtLeastOneNodeEntries(5, false, 1, 2, ""))
	// 	oneTime = true

	// case "NEntriesInAll":
	// 	machine = policies.NewRewardMachine(redisraft.AllNodesEntries(5, false, 1, 2, ""))
	// 	oneTime = true

	// at least one log with an entry in term 2
	case "EntryInTerm2[1]":
		machine = policies.NewRewardMachine(redisraft.AtLeastOneNodeEntries(1, false, 2, 2, ""))
		oneTime = true

	case "EntryInTerm2[3]":
		machine = policies.NewRewardMachine(redisraft.AtLeastOneNodeEntries(1, false, 2, 2, ""))
		machine.AddState(redisraft.AtLeastOneNodeTerm(2, 2), "OneInTerm2")
		machine.AddState(redisraft.AtLeastOneNodeStatesInTerm([]string{"leader"}, 2, 2), "LeaderInTerm2")
		oneTime = true

	// two nodes in the state "candidate"
	case "MoreThanOneCandidate":
		// Having two different candidates to enable competing elections
		machine = policies.NewRewardMachine(redisraft.NNodesInStateSameTerm(2, "candidate"))
		oneTime = true

	// two nodes in the state "leader"
	case "MoreThanOneLeader":
		// Possible if they are in different terms
		machine = policies.NewRewardMachine(redisraft.NNodesInState(2, "leader"))
		oneTime = true

	// two inconsistent logs
	case "InconsistentLogs[1]":
		machine = policies.NewRewardMachine(redisraft.InconsistentLogsPredicate())

	case "InconsistentLogs[2]":
		machine = policies.NewRewardMachine(redisraft.InconsistentLogsPredicate())
		machine.AddState(redisraft.AllNodesEntries(5, true, 1, 1, "").And(redisraft.DiffInEntries(1)), "Diff1")

		// case "NodeInDifferentTerms":
		// 	machine = policies.NewRewardMachine(redisraft.NodesInDifferentTerms(0))
		// 	oneTime = true

		// case "NodesInDifferentTermsWithLeader":
		// 	machine = policies.NewRewardMachine(redisraft.NodesInDifferentTerms(0).And(redisraft.AtLeastOneNodeStates([]string{"leader"})))
		// 	oneTime = true

		// case "MinTermDiff3":
		// 	machine = policies.NewRewardMachine(redisraft.NodesInDifferentTerms(3))
		// 	oneTime = true

		// case "MinTermDiff3WithLeader":
		// 	machine = policies.NewRewardMachine(redisraft.NodesInDifferentTerms(3).And(redisraft.AtLeastOneNodeStates([]string{"leader"})))
		// 	oneTime = true

	}

	return machine, oneTime, machine != nil
}

func RedisRaftRM(machine string, episodes, horizon int, saveFile string, ctx context.Context) {
	clusterBaseConfig := &redisraft.ClusterBaseConfig{
		NumNodes: 3,
		SavePath: saveFile,

		BasePort:            5000,
		BaseInterceptPort:   2023,
		ID:                  1,
		InterceptListenPort: 7074,

		RequestTimeout:  60,
		ElectionTimeout: 240, // new election timeout in milliseconds
		NumRequests:     3,
		TickLength:      15,
	}
	envConstructor := redisraft.RedisRaftEnvConstructor(path.Join(saveFile, "tickLength"))

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

	// chosen abstraction
	chosenColors := []string{
		"state",
		"commit",
		"leader",
		"vote",
		"boundedTerm3",
		"index",
		// "snapshot",
		// "log",
		"boundedLog3",
	}

	// create the color functions for the chosen abstraction
	colors := make([]redisraft.RedisRaftColorFunc, 0)
	for _, color := range chosenColors {
		colors = append(colors, availableColors[color])
	}

	partitionEnvConfig := types.PartitionEnvConfig{
		Ctx:                    ctx,
		Painter:                redisraft.NewRedisRaftStatePainter(colors...),
		Env:                    nil,
		EnvCtor:                envConstructor,
		EnvBaseConfig:          clusterBaseConfig,
		TicketBetweenPartition: 4,
		MaxMessagesPerTick:     100,
		StaySameStateUpto:      5,
		NumReplicas:            3,
		WithCrashes:            false,
		CrashLimit:             3,
		MaxInactive:            1,

		TerminalPredicate: redisraft.MaxTerm(3),
	}

	reportConfig := types.RepConfigStandard()
	reportConfig.SetPrintLastEpisodes(20)
	c := types.NewComparison(&types.ComparisonConfig{
		Runs:       runs,
		Episodes:   episodes,
		Horizon:    horizon,
		RecordPath: saveFile,
		Timeout:    10 * time.Second,
		// thresholds to abort the experiment
		ConsecutiveTimeoutsAbort: 20,
		ConsecutiveErrorsAbort:   20,
		// record flags
		RecordTraces: false,
		RecordTimes:  true,
		RecordPolicy: false,
		// last traces
		PrintLastTraces:     20,
		PrintLastTracesFunc: redisraft.ReadableTracePrintable,
		// report config
		ReportConfig: reportConfig,

		ParallelExperiments: 20,
		TimeBudget:          8 * time.Hour,
	})
	// after this NewComparison call, the folder is not wiped anymore

	// if flags are on, start profiling
	startProfiling()

	// c.AddAnalysis("Plot", redisraft.NewCoverageAnalyzer(horizon, colors...), redisraft.CoverageComparator(saveFile, horizon))
	c.AddAnalysis("Plot", redisraft.CoverageAnalyzerCtor(horizon, colors...), redisraft.CoverageComparator(path.Join(saveFile, "coverage"), horizon))
	c.AddAnalysis("Crashes", redisraft.BugCrashAnalyzerCtor(path.Join(saveFile, "crash")), redisraft.BugComparator())
	c.AddAnalysis("Bugs", redisraft.BugAnalyzerCtor(
		path.Join(saveFile, "bugs"),
		types.BugDesc{Name: "ReducedLog", Check: redisraft.ReducedLog()},
		types.BugDesc{Name: "ModifiedLog", Check: redisraft.ModifiedLog()},
		types.BugDesc{Name: "InconsistentLogs", Check: redisraft.InconsistentLogs()},
		// types.BugDesc{Name: "True", Check: redisraft.TruePredicate()},
		// types.BugDesc{Name: "DifferentTermsEntries", Check: redisraft.EntriesInDifferentTermsDummy()},
	), types.BugComparator(path.Join(saveFile, "bugs")))

	neg := false

	machines := getSetOfMachines(machine)
	pHierarchiesPolicies := make(map[string]*policies.RewardMachinePolicy) // map of PH policies
	for _, pHierName := range machines {                                   // for each of them create it and create an analyzer
		// special name to add negative policy to pure exploration
		if pHierName == "neg" {
			neg = true
			continue
		}

		rm, oneTime, ok := getRedisPredicateHeirarchy(pHierName)
		if !ok { // if something goes wrong just skip it
			continue
		}
		RMPolicy := policies.NewRewardMachinePolicy(rm, oneTime)
		pHierarchiesPolicies[pHierName] = RMPolicy

		// c.AddAnalysis("plot", redisraft.CoverageAnalyzer(colors...), redisraft.CoverageComparator(saveFile))
		c.AddAnalysis(pHierName, policies.RewardMachineAnalyzerCtor(RMPolicy, horizon, colors...), policies.RewardMachineCoverageComparator(path.Join(saveFile, "coverage"), pHierName))
	}

	for pHierName, policy := range pHierarchiesPolicies {
		c.AddExperiment(types.NewExperiment(pHierName, policy, types.NewPartitionEnv(partitionEnvConfig)))
	}

	baselines := true

	if len(machines) > 0 { // if there are machines to run, check if baselines are included
		baselines = machines[len(machines)-1] == "baselines"
	}

	if baselines {
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
		if neg {
			c.AddExperiment(types.NewExperiment(
				"negVisits",
				policies.NewSoftMaxNegFreqPolicy(0.3, 0.7, 1),
				types.NewPartitionEnv(partitionEnvConfig),
			))
			c.AddExperiment(types.NewExperiment(
				"neg",
				types.NewSoftMaxNegPolicy(0.1, 0.99, 1),
				types.NewPartitionEnv(partitionEnvConfig),
			))
			c.AddExperiment(types.NewExperiment(
				"negHT",
				types.NewSoftMaxNegPolicy(0.1, 0.99, 10),
				types.NewPartitionEnv(partitionEnvConfig),
			))
		}

	}

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
	util.WriteToFile(configPath, ExperimentParametersPrintable(), clusterBaseConfig.Printable(), partitionEnvConfig.Printable(), PrintColors(chosenColors))

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
