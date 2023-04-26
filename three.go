package main

import (
	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/raft"
	"github.com/zeu5/raft-rl-test/types"
)

func Three(episodes, horizon int, saveFile string) {
	raftConfig := raft.RaftEnvironmentConfig{
		Replicas:      3,
		ElectionTick:  10,
		HeartbeatTick: 1,
		Timeouts:      true,
		TicksPerStep:  2,
	}
	c := types.NewComparison(raft.RaftAnalyzer(saveFile), raft.RaftPlotComparator(saveFile, raft.ChainFilters(raft.MinCutOff(100), raft.Log())))
	c.AddExperiment(types.NewExperiment("RL", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      types.NewSoftMaxNegPolicy(0.3, 0.7),
		Environment: raft.NewRaftEnvironment(raftConfig),
	}))
	c.AddExperiment(types.NewExperiment("Random", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      types.NewRandomPolicy(),
		Environment: raft.NewRaftEnvironment(raftConfig),
	}))
	c.AddExperiment(types.NewExperiment("BonusMaxRL", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      policies.NewBonusPolicyGreedy(horizon, 0.99, true),
		Environment: raft.NewRaftEnvironment(raftConfig),
	}))

	c.Run()
}

func ThreeCommand() *cobra.Command {
	return &cobra.Command{
		Use: "three",
		Run: func(cmd *cobra.Command, args []string) {
			Three(episodes, horizon, saveFile)
		},
	}
}

// Play with level of abstraction in state space
// 1. Less/more information effect
// 2. Effect of timeouts
// 3. Better toy example
// 4. Visualization of the paxos example to see the state space
// 5. Try a synchronous example
