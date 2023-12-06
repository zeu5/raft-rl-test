package benchmarks

import (
	"context"
	"os"
	"os/signal"
	"path"

	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/redisraft"
	"github.com/zeu5/raft-rl-test/types"
)

func getRedisPredicateHeirarchy(name string) (*policies.RewardMachine, bool) {
	var machine *policies.RewardMachine = nil
	switch name {
	case "OnlyFollowersAndLeader":
		// This is always true initially
		machine = policies.NewRewardMachine(redisraft.OnlyFollowersAndLeader())
	case "ElectLeader":
		// This is also always true. The system starts with a config where the leader is elected
		machine = policies.NewRewardMachine(redisraft.LeaderElected())
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
	}
	return machine, machine != nil
}

func RedisRaftRM(machine string, episodes, horizon int, saveFile string, ctx context.Context) {
	clusterConfig := redisraft.ClusterConfig{
		NumNodes:            3,
		BasePort:            5000,
		BaseInterceptPort:   2023,
		ID:                  1,
		InterceptListenAddr: "localhost:7074",
		WorkingDir:          path.Join(saveFile, "tmp"),
		NumRequests:         3,

		RequestTimeout:  25,  // heartbeat in milliseconds (fixed or variable?)
		ElectionTimeout: 120, // election timeout in milliseconds (from specified value to its double)

		TickLength: 20,
	}
	env := redisraft.NewRedisRaftEnv(ctx, &clusterConfig, path.Join(saveFile, "tickLength"))
	defer env.Cleanup()

	// abstraction for both plot and RL
	availableColors := make(map[string]redisraft.RedisRaftColorFunc)
	availableColors["state"] = redisraft.ColorState()                 // replica internal state
	availableColors["commit"] = redisraft.ColorCommit()               // number of committed entries? includes config changes, leader elect, request entry
	availableColors["leader"] = redisraft.ColorLeader()               // if a replica is leader? boolean?
	availableColors["vote"] = redisraft.ColorVote()                   // ?
	availableColors["index"] = redisraft.ColorIndex()                 // next available index to write?
	availableColors["boundedTerm5"] = redisraft.ColorBoundedTerm(5)   // current term, bounded to the passed value
	availableColors["boundedTerm10"] = redisraft.ColorBoundedTerm(10) // current term, bounded to the passed value
	availableColors["applied"] = redisraft.ColorApplied()
	availableColors["snapshot"] = redisraft.ColorSnapshot()

	chosenColors := []string{
		"state",
		"commit",
		"leader",
		"vote",
		"boundedTerm10",
		"index",
		"snapshot",
	}

	colors := make([]redisraft.RedisRaftColorFunc, 0)
	for _, color := range chosenColors {
		colors = append(colors, availableColors[color])
	}

	// colors := []redisraft.RedisRaftColorFunc{redisraft.ColorState(), redisraft.ColorCommit(), redisraft.ColorLeader(), redisraft.ColorVote(), redisraft.ColorBoundedTerm(5), redisraft.ColorIndex(), redisraft.ColorSnapshot()}

	partitionEnvConfig := types.PartitionEnvConfig{
		Painter:                redisraft.NewRedisRaftStatePainter(colors...),
		Env:                    env,
		TicketBetweenPartition: 3,
		MaxMessagesPerTick:     100,
		StaySameStateUpto:      5,
		NumReplicas:            3,
		WithCrashes:            true,
		CrashLimit:             10,
		MaxInactive:            0,
	}

	c := types.NewComparison(runs, saveFile, false)

	c.AddEAnalysis(types.RecordPartitionStats(saveFile))

	c.AddAnalysis("Plot", redisraft.CoverageAnalyzer(colors...), redisraft.CoverageComparator(saveFile))
	c.AddAnalysis("Crashes", redisraft.BugAnalyzerCrash(path.Join(saveFile, "crash")), redisraft.BugComparator())
	c.AddAnalysis("Bugs", redisraft.BugAnalyzer(
		path.Join(saveFile, "bugs"),
		types.BugDesc{Name: "ReducedLog", Check: redisraft.ReducedLog()},
		types.BugDesc{Name: "ModifiedLog", Check: redisraft.ModifiedLog()},
		types.BugDesc{Name: "InconsistentLogs", Check: redisraft.InconsistentLogs()},
		types.BugDesc{Name: "True", Check: redisraft.TruePredicate()},
	), types.BugComparator(path.Join(saveFile, "bugs")))
	rm, ok := getRedisPredicateHeirarchy(machine)
	if !ok {
		return
	}
	RMPolicy := policies.NewRewardMachinePolicy(rm, false)

	// c.AddAnalysis("plot", redisraft.CoverageAnalyzer(colors...), redisraft.CoverageComparator(saveFile))
	c.AddAnalysis("rm", policies.RewardMachineAnalyzer(RMPolicy), policies.RewardMachineCoverageComparator(saveFile))

	c.AddExperiment(types.NewExperiment("predHierarchy", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      RMPolicy,
		Environment: types.NewPartitionEnv(partitionEnvConfig),
	}))
	c.AddExperiment(types.NewExperiment("random", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      types.NewRandomPolicy(),
		Environment: types.NewPartitionEnv(partitionEnvConfig),
	}))
	c.AddExperiment(types.NewExperiment("rl", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      policies.NewBonusPolicyGreedy(0.1, 0.99, 0.05),
		Environment: types.NewPartitionEnv(partitionEnvConfig),
	}))

	// print config file
	configPath := path.Join(saveFile, "config.txt")
	types.WriteToFile(configPath, clusterConfig.Printable(), partitionEnvConfig.Printable(), PrintColors(chosenColors))

	c.Run()
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
