package main

import (
	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/raft"
	"github.com/zeu5/raft-rl-test/rl"
)

func One(episodes, horizon int, saveFile string) {
	raftConfig := raft.RaftEnvironmentConfig{
		Replicas:      3,
		ElectionTick:  10,
		HeartbeatTick: 1,
		Timeouts:      true,
	}
	c := rl.NewComparison(raft.RaftAnalyzer, raft.RaftPlotComparator(saveFile))
	c.AddExperiment(rl.NewExperiment("RL", &rl.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      rl.NewSoftMaxNegPolicy(0.3, 0.7),
		Environment: raft.NewRaftEnvironment(raftConfig),
	}))
	c.AddExperiment(rl.NewExperiment("Random", &rl.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      rl.NewRandomPolicy(),
		Environment: raft.NewRaftEnvironment(raftConfig),
	}))
	raftConfig.Timeouts = false
	c.AddExperiment(rl.NewExperiment("RL-NoTimeouts", &rl.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      rl.NewSoftMaxNegPolicy(0.3, 0.7),
		Environment: raft.NewRaftEnvironment(raftConfig),
	}))
	c.AddExperiment(rl.NewExperiment("Random-NoTimeouts", &rl.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      rl.NewRandomPolicy(),
		Environment: raft.NewRaftEnvironment(raftConfig),
	}))

	c.Run()
}

func OneCommand() *cobra.Command {
	return &cobra.Command{
		Use: "one",
		Run: func(cmd *cobra.Command, args []string) {
			One(episodes, horizon, saveFile)
		},
	}
}
