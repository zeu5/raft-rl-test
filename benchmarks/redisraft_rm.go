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
		return []string{"SyncA6_PR3_MT1_>>_CReq1_PR2", "SA6_PR3_MT1_>>_C1T1_E1T2_PR1", "SA6_PR3_MT1_>>_2CommitsInDiffTerms"}
	case "Set5":
		return []string{"2CommT1X", "LogDiff3_Steps", "CommT1_LeadT25_LogDiff3", "baselines"}
	case "Set6":
		return []string{"OneTerm2", "LeaderInTerm2", "AllInTerm2", "AllInTerm3", "baselines"}
	case "Set7":
		return []string{"2CommT1X", "LogDiff3_Steps", "CommT1_LeadT25_LogDiff3", "OneTerm2", "LeaderInTerm2", "AllInTerm2", "AllInTerm3", "baselines"}
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

	// Sync all nodes to 6 generic entries, with 3 pending requests
	case "SyncA6_PR3_MT1_>>_CReq1_PR2":
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.AllNodesEntries(3, true, 1, 1, "").And(redisraft.PendingRequestsAtLeast(3)), Name: "SyncAll3PReq3MT1"})
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.AllNodesEntries(4, true, 1, 1, "").And(redisraft.PendingRequestsAtLeast(3)), Name: "SyncAll4PReq3MT1"})
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.AllNodesEntries(5, true, 1, 1, "").And(redisraft.PendingRequestsAtLeast(3)), Name: "SyncAll5PReq3MT1"})
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.AllNodesEntries(6, true, 1, 1, "").And(redisraft.PendingRequestsAtLeast(2)).And(redisraft.AllNodesEntries(1, true, 1, 1, "NORMAL")), Name: "Comm1PReq2MT1"})

	case "InitSync_C1T1_T2_PReq2":
		RedisPredHierAddBuildBlocks(pHier, "SyncA6_PR3_MT1_>>_CReq1_PR2")
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.AllNodesEntries(6, true, 1, 1, "").
			And(redisraft.PendingRequestsAtLeast(2)).
			And(redisraft.AllNodesEntries(1, true, 1, 1, "NORMAL")).
			And(redisraft.AllNodesTerms(1, 2)).
			And(redisraft.AtLeastOneNodeTerm(2, 2)), Name: "Sync_C_T2_PR2"})

	case "Sync_C1T1_E1T2":
		RedisPredHierAddBuildBlocks(pHier, "SyncA5_MT1") // sync to 5 with 3 pending requests (initial registration)
		Req1T1 := redisraft.AllNodesEntries(5, true, 1, 1, "").And(redisraft.AtLeastOneNodeEntries(1, false, 1, 1, "NORMAL")).And((redisraft.AtLeastOneNodeEntries(2, false, 1, 1, "NORMAL").Not()))
		toAdd = append(toAdd, predHierState{RewFunc: Req1T1, Name: "Req1T1"})
		// commit 1 request
		Commit1T1 := redisraft.AllNodesEntries(6, true, 1, 1, "").And(redisraft.AllNodesEntries(1, true, 1, 1, "NORMAL")).And((redisraft.AtLeastOneNodeEntries(2, false, 1, 1, "NORMAL").Not()))
		toAdd = append(toAdd, predHierState{RewFunc: Commit1T1, Name: "Commit1T1"})
		// at least one process in term2
		AtLeastOneInT2 := redisraft.AllNodesTerms(1, 2).And(redisraft.AtLeastOneNodeTerm(2, 2))
		toAdd = append(toAdd, predHierState{RewFunc: AtLeastOneInT2.And(Commit1T1), Name: "C1T1_AtLeastOneInT2"})
		// all processes in term2
		AllInT2 := redisraft.AllNodesTerms(2, 2)
		toAdd = append(toAdd, predHierState{RewFunc: AllInT2.And(Commit1T1), Name: "C1T1_AllInT2"})
		// all in T2 and one leader
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.AtLeastOneNodeStates([]string{"leader"}).And(AllInT2.And(Commit1T1)), Name: "C1T1_AllInT2_OneLeader"})
		// at least one entry in T2
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.AtLeastOneNodeEntries(1, false, 2, 2, "").And(Commit1T1).And(AllInT2), Name: "C1T1_AllInT2_OneE1T2"})
		// all one entry in T2
		C1T1_AllInT2_E1T2 := redisraft.AllNodesEntries(1, false, 2, 2, "").And(Commit1T1).And(AllInT2)
		toAdd = append(toAdd, predHierState{RewFunc: C1T1_AllInT2_E1T2, Name: "C1T1_AllInT2_E1T2"})
		// commit 1 request in T2
		// toAdd = append(toAdd, predHierState{RewFunc: redisraft.AllNodesEntries(1, true, 2, 2, "NORMAL").And(C1T1_AllInT2_E1T2), Name: "C1T1_AllInT2_E1T2_PReq1"})

	case "Sync_C1T1_LeaderX":
		RedisPredHierAddBuildBlocks(pHier, "SyncA5_MT1") // sync to 5 with 3 pending requests (initial registration)
		Req1T1 := redisraft.AllNodesEntries(5, true, 1, 1, "").And(redisraft.AtLeastOneNodeEntries(1, false, 1, 1, "NORMAL")).And((redisraft.AtLeastOneNodeEntries(3, false, 1, 1, "NORMAL").Not()))
		toAdd = append(toAdd, predHierState{RewFunc: Req1T1, Name: "Req1T1"})
		// commit 1 request
		Commit1T1 := redisraft.AllNodesEntries(6, true, 1, 1, "").And(redisraft.AllNodesEntries(1, true, 1, 1, "NORMAL")).And((redisraft.AtLeastOneNodeEntries(3, false, 1, 1, "NORMAL").Not()))
		toAdd = append(toAdd, predHierState{RewFunc: Commit1T1, Name: "Commit1T1"})
		// at least one process in term2
		AtLeastOneInT2 := redisraft.AllNodesTerms(1, 5).And(redisraft.AtLeastOneNodeTerm(2, 5))
		toAdd = append(toAdd, predHierState{RewFunc: AtLeastOneInT2.And(Commit1T1), Name: "C1T1_AtLeastOneInTX"})
		toAdd = append(toAdd, predHierState{RewFunc: AtLeastOneInT2.And(Commit1T1).And(redisraft.AllNodesTerms(2, 2).Or(redisraft.AllNodesTerms(3, 3).Or(redisraft.AllNodesTerms(4, 4)).Or(redisraft.AllNodesTerms(5, 5)))), Name: "C1T1_AllInSameT25"})
		// all in T2 and one leader
		toAdd = append(toAdd, predHierState{RewFunc: redisraft.AtLeastOneNodeStates([]string{"leader"}).And(redisraft.AllNodesTerms(2, 5).And(Commit1T1)), Name: "C1T1_AllInT2X_Leader"})

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
		RedisPredHierAddBuildBlocks(machine, "SyncA5_PR2_MT1")
		oneTime = true

	case "SA6_PR3_MT1_>>_C1T1_E1T2_PR1":
		machine = policies.NewRewardMachine(redisraft.AllNodesEntries(6, true, 1, 1, "").And(redisraft.PendingRequestsAtLeast(1)).And(redisraft.AllNodesEntries(1, true, 1, 1, "NORMAL")).And(redisraft.AtLeastOneNodeEntries(1, false, 2, 2, "")))
		RedisPredHierAddBuildBlocks(machine, "InitSync_C1T1_T2_PReq2")
		oneTime = true

	case "SA6_PR3_MT1_>>_2CommitsInDiffTerms":
		machine = policies.NewRewardMachine(redisraft.AllNodesEntries(6, true, 1, 1, "").And(redisraft.AllNodesEntries(1, true, 1, 1, "NORMAL")).And(redisraft.AllNodesEntries(1, true, 2, 10, "NORMAL")))
		RedisPredHierAddBuildBlocks(machine, "SyncA6_PR3_MT1_>>_CReq1_PR2")
		oneTime = true

	case "2CommT12_easy":
		machine = policies.NewRewardMachine(redisraft.AllNodesEntries(6, true, 1, 1, "").
			And(redisraft.AllNodesEntries(1, true, 1, 1, "NORMAL")).
			And(redisraft.AllNodesTerms(2, 2)).
			// And(redisraft.AllNodesEntries(2, true, 2, 2, "")).
			And(redisraft.AllNodesEntries(1, true, 2, 2, "NORMAL")))
		RedisPredHierAddBuildBlocks(machine, "Sync_C1T1_E1T2")
		oneTime = true

	case "2CommT1X":
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

	case "LogDiff3":
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
		machine = policies.NewRewardMachine(redisraft.AllNodesEntries(5, true, 1, 1, "").
			And(redisraft.DiffInCommittedEntries(3)))
		RedisPredHierAddBuildBlocks(machine, "SyncA5_MT1_NoNormal")
		machine.AddState(redisraft.AllNodesEntries(5, true, 1, 1, "").And(redisraft.DiffInEntries(1)), "Diff1")
		machine.AddState(redisraft.AllNodesEntries(5, true, 1, 1, "").And(redisraft.DiffInEntries(2)), "Diff2")
		machine.AddState(redisraft.AllNodesEntries(5, true, 1, 1, "").And(redisraft.DiffInEntries(3)), "Diff3")
		oneTime = true

	case "LogDiff4_Steps":
		machine = policies.NewRewardMachine(redisraft.AllNodesEntries(5, true, 1, 1, "").
			And(redisraft.DiffInCommittedEntries(4)))
		RedisPredHierAddBuildBlocks(machine, "SyncA5_MT1_NoNormal")
		machine.AddState(redisraft.AllNodesEntries(5, true, 1, 1, "").And(redisraft.DiffInEntries(1)), "Diff1")
		machine.AddState(redisraft.AllNodesEntries(5, true, 1, 1, "").And(redisraft.DiffInEntries(2)), "Diff2")
		machine.AddState(redisraft.AllNodesEntries(5, true, 1, 1, "").And(redisraft.DiffInEntries(3)), "Diff3")
		machine.AddState(redisraft.AllNodesEntries(5, true, 1, 1, "").And(redisraft.DiffInEntries(4)), "Diff4")
		oneTime = true

	case "CommT1_LeadT25_LogDiff3":
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
		machine = policies.NewRewardMachine(redisraft.AtLeastOneNodeTerm(2, 2))
		oneTime = true

	case "AllInTerm2":
		machine = policies.NewRewardMachine(redisraft.AllNodesTerms(2, 2))
		oneTime = true

	case "AllInTerm3":
		machine = policies.NewRewardMachine(redisraft.AllNodesTerms(3, 3))
		oneTime = true

	case "LeaderInTerm2":
		machine = policies.NewRewardMachine(redisraft.AtLeastOneNodeStates([]string{"leader"}).And(redisraft.AtLeastOneNodeTerm(2, 2)))
		oneTime = true
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

		RequestTimeout:  100,
		ElectionTimeout: 400, // new election timeout in milliseconds
		NumRequests:     5,
		TickLength:      25,
	}
	envConstructor := redisraft.RedisRaftEnvConstructor(path.Join(saveFile, "tickLength"))

	// clusterConfig := redisraft.ClusterConfig{
	// 	NumNodes:            3,
	// 	BasePort:            5000,
	// 	BaseInterceptPort:   2023,
	// 	ID:                  1,
	// 	InterceptListenAddr: "localhost:7074",
	// 	WorkingDir:          path.Join(saveFile, "tmp"),
	// 	NumRequests:         5,

	// 	RequestTimeout:  100, // heartbeat in milliseconds (fixed or variable?)
	// 	ElectionTimeout: 400, // election timeout in milliseconds (from specified value to its double)

	// 	TickLength: 25,
	// }
	// env := redisraft.NewRedisRaftEnv(ctx, &clusterConfig, path.Join(saveFile, "tickLength"))
	// env.SetPrintStats(true) // to print the episode stats
	// defer env.Cleanup()

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
		Ctx:                    ctx,
		Painter:                redisraft.NewRedisRaftStatePainter(colors...),
		Env:                    nil,
		EnvCtor:                envConstructor,
		EnvBaseConfig:          clusterBaseConfig,
		TicketBetweenPartition: 4,
		MaxMessagesPerTick:     100,
		StaySameStateUpto:      5,
		NumReplicas:            3,
		WithCrashes:            true,
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
		ConsecutiveTimeoutsAbort: 10,
		ConsecutiveErrorsAbort:   10,
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
	})
	// after this NewComparison call, the folder is not wiped anymore

	// if flags are on, start profiling
	startProfiling()

	// c.AddAnalysis("Plot", redisraft.NewCoverageAnalyzer(horizon, colors...), redisraft.CoverageComparator(saveFile, horizon))
	c.AddAnalysis("Plot", redisraft.CoverageAnalyzerCtor(horizon, colors...), redisraft.CoverageComparator(saveFile, horizon))
	c.AddAnalysis("Crashes", redisraft.BugCrashAnalyzerCtor(path.Join(saveFile, "crash")), redisraft.BugComparator())
	c.AddAnalysis("Bugs", redisraft.BugAnalyzerCtor(
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
		c.AddAnalysis(pHierName, policies.RewardMachineAnalyzerCtor(RMPolicy), policies.RewardMachineCoverageComparator(saveFile, pHierName))
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
