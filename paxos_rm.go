package main

import (
	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/lpaxos"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/types"
)

func PaxosRewardMachine(episodes, horizon int) {
	lPaxosConfig := lpaxos.LPaxosEnvConfig{
		Replicas: 3,
		Requests: requests,
		Timeout:  12,
		Timeouts: timeouts,
	}

	commit := lpaxos.Commit()
	allPredicates := []types.RewardFunc{commit}

	rm := policies.NewRewardMachine(nil)
	// rm.On(commit, "commit")

	c := types.NewComparison(policies.RewardMachineAnalyzer(allPredicates, lpaxos.PaxosStateAbstractor()), policies.RewardMachineCoverageComparator())
	c.AddExperiment(types.NewExperiment(
		"Random-Part",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      types.NewRandomPolicy(),
			Environment: getLPaxosPartEnv(lPaxosConfig, true),
		},
	))
	c.AddExperiment(types.NewExperiment(
		"BonusMaxRL",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      policies.NewBonusPolicyGreedy(0.1, 0.99, 0.2),
			Environment: getLPaxosEnv(lPaxosConfig, abstracter),
		},
	))
	c.AddExperiment(types.NewExperiment(
		"RewardMachine",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      policies.NewRewardMachinePolicy(rm),
			Environment: getLPaxosPartEnv(lPaxosConfig, true),
		},
	))
	c.Run()
}

func PaxosRewardMachineCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "paxos-reward-rm",
		Run: func(cmd *cobra.Command, args []string) {
			PaxosRewardMachine(episodes, horizon)
		},
	}
	cmd.PersistentFlags().IntVarP(&requests, "requests", "r", 1, "Number of requests to run with")
	cmd.PersistentFlags().BoolVarP(&timeouts, "timeouts", "t", false, "Run with timeouts or not")
	return cmd
}
