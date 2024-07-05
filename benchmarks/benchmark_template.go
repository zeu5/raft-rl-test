package benchmarks

import (
	"context"
	"fmt"
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

// return the list of predicates sequences to use in the experiment, returns empty list if unknown value
// each Predicate Sequence is a separated experiment
func getPredicatesSeqSet(command string) []string {
	switch command {
	// case ...:
	// return ["seq1", ..., "seqN"]
	default:
		return []string{}
	}
}

// here the predicates sequences can be defined, each of them should be identified by a name (string)
func getPredicateSequence(name string, rmRlConfig policies.RMRLConfig) (*policies.RewardMachine, bool, bool) {
	var machine *policies.RewardMachine = nil
	oneTime := false
	switch name {

	// case "EXAMPLE_NAME":
	// machine = policies.NewRewardMachine(FINAL_PREDICATE, rmRlConfig)
	// machine.AddState(WAYPOINT_1, "WAYPOINT_1_NAME")
	// machine.AddState(WAYPOINT_2, "WAYPOINT_2_NAME")
	// oneTime = true/false

	// concrete example with RedisRaft predicates
	case "EntryInTerm2[3]":
		machine = policies.NewRewardMachine(redisraft.AtLeastOneNodeEntries(1, false, 2, 2, ""), rmRlConfig)
		machine.AddState(redisraft.AtLeastOneNodeTerm(2, 2).And(redisraft.PendingRequestsAtLeast(2)).And(redisraft.AllNodesTerms(1, 2)), "OneInTerm2")
		machine.AddState(redisraft.AtLeastOneNodeStatesInTerm([]string{"leader"}, 2, 2).And(redisraft.PendingRequestsAtLeast(2)).And(redisraft.AllNodesTerms(1, 2)), "LeaderInTerm2")
		oneTime = true

	}

	return machine, oneTime, machine != nil
}

func RunBenchmark(sequencesSet string, episodes, horizon int, saveFile string, ctx context.Context) {

	// benchmark configuration
	numberOfNodes := 3
	clusterBaseConfig := &redisraft.ClusterBaseConfig{
		// number of nodes and output path
		NumNodes: numberOfNodes,
		SavePath: saveFile,

		// message ports to use
		BasePort:            5000,
		BaseInterceptPort:   2023,
		ID:                  1,
		InterceptListenPort: 7074,

		// raft-specific config, nodes timeouts
		RequestTimeout:  50,  // milliseconds
		ElectionTimeout: 200, // new election timeout in milliseconds
		NumRequests:     3,   // number of client requests
		TickLength:      25,  // milliseconds
	}
	envConstructor := redisraft.RedisRaftEnvConstructor(saveFile) // call the specific function to get the underlying environment constructor of that benchmark

	// define parts of state abstractions
	availableColors := make(map[string]redisraft.RedisRaftColorFunc)
	// availableColors["NAME"] = BENCHMARK_STATE_INFO_EXTRACT_FUNCTION
	availableColors["state"] = redisraft.ColorState()   // replica internal state
	availableColors["commit"] = redisraft.ColorCommit() // number of committed entries in a replica's log
	availableColors["leader"] = redisraft.ColorLeader()
	availableColors["boundedLog3"] = redisraft.ColorBoundedLog(3)
	availableColors["boundedLog5"] = redisraft.ColorBoundedLog(5)

	// specify which parts will be considered to define colors (state abstractions), everything else will be ignored
	chosenColors := []string{
		// "NAME"
		"state",
		"commit",
		"boundedLog5",
	}

	// create the color functions for the chosen abstraction
	colors := make([]redisraft.RedisRaftColorFunc, 0)
	for _, color := range chosenColors {
		colors = append(colors, availableColors[color])
	}

	// generic partition environment configuration
	partitionEnvConfig := types.PartitionEnvConfig{
		Ctx:           ctx,
		Painter:       redisraft.NewRedisRaftStatePainter(colors...), // abstraction function using the chosen colors (benchmark-specific)
		Env:           nil,
		EnvCtor:       envConstructor,    // underlying environment constructor (benchmark-specific)
		EnvBaseConfig: clusterBaseConfig, // underlying environment config (benchmark-specific)

		// ticks, messages, ...
		TicketBetweenPartition: 4,             // how many ticks between two actions (partitions)
		MaxMessagesPerTick:     100,           // max amount of delivered messages per tick (nodes in the same partition)
		StaySameStateUpto:      5,             // max value of consecutive steps in the same configuration
		NumReplicas:            numberOfNodes, // number of nodes

		// crashes configuration
		WithCrashes: true, // enabled
		CrashLimit:  3,    // max number of nodes crashed per episode
		MaxInactive: 1,    // max amount of simultaneous inactive nodes

		// search space bound, episodes will end when out of bound
		TerminalPredicate:            redisraft.MaxTerm(5), // constant false predicate to not set any bound
		TerminalPredicateDescription: "MaxTerm(5)",
	}

	// additional execution information, ticks timings, number of messages, episodes outcomes and traces...
	// stored in the output folder, can be used to check learned policy
	reportConfig := types.RepConfigStandard()
	reportConfig.SetPrintLastEpisodes(20)

	comparisonTimeBudget := 8 * time.Hour
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
		PrintLastTracesTime: time.Duration(5 * time.Minute),
		PrintLastTracesFunc: redisraft.ReadableTracePrintable,
		// report config
		ReportConfig: reportConfig,

		ParallelExperiments: 20,
		TimeBudget:          comparisonTimeBudget,
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

	RmRlConfigs := make([]policies.RMRLConfig, 0)
	RmRlConfigs = append(RmRlConfigs, policies.RMRLConfig{
		LearningRate: 0.2,
		Discount:     0.95,
		Epsilon:      0.05,
	})
	// RmRlConfigs = append(RmRlConfigs, policies.RMRLConfig{
	// 	LearningRate: 0.3,
	// 	Discount:     0.90,
	// 	Epsilon:      0.05,
	// })

	sequences := getSetOfMachines(sequencesSet)
	pHierarchiesPolicies := make(map[string]*policies.RewardMachinePolicy) // map of PH policies
	for _, pHierName := range sequences {                                  // for each of them create it and create an analyzer
		// special name to add negative policy to pure exploration
		if pHierName == "neg" {
			neg = true
			continue
		}

		var RMPolicy *policies.RewardMachinePolicy

		// multile Reward Machine policy parameters to test
		if len(RmRlConfigs) > 1 {
			// add one for each config and append the cfg number to the name
			for i, rlConfig := range RmRlConfigs {
				rm, oneTime, ok := getRedisPredicateHeirarchy(pHierName, rlConfig)
				if !ok { // if something goes wrong just skip it
					continue
				}
				pHNameCfg := pHierName + "-cfg" + fmt.Sprintf("%d", i)
				RMPolicy = policies.NewRewardMachinePolicy(rm, oneTime)
				pHierarchiesPolicies[pHNameCfg] = RMPolicy
			}
		} else { // single Reward Machine policy parameters
			rm, oneTime, ok := getRedisPredicateHeirarchy(pHierName, RmRlConfigs[0])
			if !ok { // if something goes wrong just skip it
				continue
			}
			RMPolicy = policies.NewRewardMachinePolicy(rm, oneTime)
			pHierarchiesPolicies[pHierName] = RMPolicy
		}

		// c.AddAnalysis("plot", redisraft.CoverageAnalyzer(colors...), redisraft.CoverageComparator(saveFile))
		c.AddAnalysis(pHierName, policies.RewardMachineAnalyzerCtor(RMPolicy, horizon, colors...), policies.RewardMachineCoverageComparator(path.Join(saveFile, "coverage"), pHierName))
	}

	for pHierName, policy := range pHierarchiesPolicies {
		c.AddExperiment(types.NewExperiment(pHierName, policy, types.NewPartitionEnv(partitionEnvConfig)))
	}

	baselines := true

	if len(sequences) > 0 { // if there are machines to run, check if baselines are included
		baselines = sequences[len(sequences)-1] == "baselines"
	}

	if baselines {
		c.AddExperiment(types.NewExperiment(
			"bonusRlMax",
			policies.NewBonusPolicyGreedy(0.2, 0.95, 0.05, true),
			types.NewPartitionEnv(partitionEnvConfig),
		))
		c.AddExperiment(types.NewExperiment(
			"random",
			types.NewRandomPolicy(),
			types.NewPartitionEnv(partitionEnvConfig),
		))
		c.AddExperiment(types.NewExperiment(
			"negVisits",
			policies.NewSoftMaxNegFreqPolicy(0.3, 0.7, 1, false),
			types.NewPartitionEnv(partitionEnvConfig),
		))
		if neg {
			c.AddExperiment(types.NewExperiment(
				"neg",
				types.NewSoftMaxNegPolicy(0.1, 0.99, 1),
				types.NewPartitionEnv(partitionEnvConfig),
			))
			c.AddExperiment(types.NewExperiment(
				"bonusRl",
				policies.NewBonusPolicyGreedy(0.1, 0.99, 0.05, false),
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
	util.WriteToFile(configPath, ExperimentParametersPrintable(), clusterBaseConfig.Printable(), partitionEnvConfig.Printable(), PrintColors(chosenColors), policies.RMRLConfigsListPrintable(RmRlConfigs), fmt.Sprintf("Time Budget: %s", comparisonTimeBudget.String()))

	c.Run(ctx)
}

func BenchmarkCommand() *cobra.Command {
	return &cobra.Command{
		Use:  "BENCHMARK_NAME [SEQUENCES_SET]", //
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

			RunBenchmark(args[0], episodes, horizon, saveFile, ctx)

			close(doneCh)
		},
	}
}
