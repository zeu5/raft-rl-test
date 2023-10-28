package main

// experiment file

import (
	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/raft"
	"github.com/zeu5/raft-rl-test/types"
)

func RaftPart(episodes, horizon int, saveFile string) {
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

	// c is general experiment
	// colors ... , expanded list, can omit the argument
	// Analyzer takes the path to save data and colors... is the abstraction used to plot => makes the datasets
	// PlotComparator => makes plots from data
	c := types.NewComparison(runs)

	// here you add different traces analysis and comparators -- to process traces into a dataset (analyzer) and output the results (comparator)
	c.AddAnalysis("Plot", raft.RaftAnalyzer(saveFile, colors...), raft.RaftPlotComparator(saveFile))

	// here you add different policies with their parameters
	c.AddExperiment(types.NewExperiment("RL", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      policies.NewSoftMaxNegFreqPolicy(0.3, 0.7, 1),
		Environment: getRaftPartEnv(raftConfig, colors),
	}))
	c.AddExperiment(types.NewExperiment("Random", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      types.NewRandomPolicy(),
		Environment: getRaftPartEnv(raftConfig, colors),
	}))
	c.AddExperiment(types.NewExperiment("BonusMaxRL", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      policies.NewBonusPolicyGreedy(0.1, 0.99, 0.2),
		Environment: getRaftPartEnv(raftConfig, colors),
	}))

	c.Run()
}

func getRaftPartEnv(config raft.RaftEnvironmentConfig, colors []raft.RaftColorFunc) types.Environment {

	return types.NewPartitionEnv(types.PartitionEnvConfig{
		Painter:                raft.NewRaftStatePainter(colors...),  // pass the abstraction to env
		Env:                    raft.NewPartitionEnvironment(config), // actual environment
		TicketBetweenPartition: 3,                                    // ticks between actions
		MaxMessagesPerTick:     3,                                    // upper bound of random num of delivered messages
		StaySameStateUpto:      2,                                    // counter to distinguish consecutive states
		NumReplicas:            config.Replicas,
	})
}

func RaftPartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "raft-part",
		Run: func(cmd *cobra.Command, args []string) {
			RaftPart(episodes, horizon, saveFile)
		},
	}
	cmd.PersistentFlags().IntVarP(&requests, "requests", "r", 1, "Number of requests to run with")
	return cmd
}
