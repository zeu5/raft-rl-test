package main

import (
	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/lpaxos"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/types"
)

func Paxos(episodes, horizon int, saveFile string) {
	lPaxosConfig := lpaxos.LPaxosEnvConfig{
		Replicas: 3,
		Requests: 2,
		Timeout:  12,
		Timeouts: true,
	}
	c := types.NewComparison(lpaxos.LPaxosAnalyzer(saveFile), lpaxos.LPaxosComparator(saveFile))
	c.AddExperiment(types.NewExperiment("DeliverAll", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      lpaxos.NewOnlyDeliverPolicy(),
		Environment: lpaxos.NewLPaxosEnv(lPaxosConfig),
	}))
	c.AddExperiment(types.NewExperiment("RL", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      types.NewSoftMaxNegPolicy(0.3, 0.7),
		Environment: lpaxos.NewLPaxosEnv(lPaxosConfig),
	}))
	c.AddExperiment(types.NewExperiment("Random", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      types.NewRandomPolicy(),
		Environment: lpaxos.NewLPaxosEnv(lPaxosConfig),
	}))
	c.AddExperiment(types.NewExperiment("BonusMaxRL", &types.AgentConfig{
		Episodes:    episodes,
		Horizon:     horizon,
		Policy:      policies.NewBonusPolicyGreedy(horizon, 0.99, true),
		Environment: lpaxos.NewLPaxosEnv(lPaxosConfig),
	}))

	c.Run()
}

func PaxosCommand() *cobra.Command {
	return &cobra.Command{
		Use: "paxos",
		Run: func(cmd *cobra.Command, args []string) {
			Paxos(episodes, horizon, saveFile)
		},
	}
}
