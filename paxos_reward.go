package main

import (
	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/lpaxos"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/types"
)

func PaxosReward(episodes, horizon int, saveFile string) {
	lPaxosConfig := lpaxos.LPaxosEnvConfig{
		Replicas: 3,
		Requests: requests,
		Timeout:  12,
		Timeouts: timeouts,
	}

	inPrepare := lpaxos.InPhase(lpaxos.StepPropose)

	c := types.NewComparison(lpaxos.RewardStatesVisitedAnalyzer([]string{"commit"}, []types.RewardFunc{inPrepare}, saveFile), lpaxos.RewardStateComparator())
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
		"Biased-Policy",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      policies.NewGuidedPolicy(inPrepare, 0.2, 0.95, 0.02),
			Environment: getLPaxosPartEnv(lPaxosConfig, true),
		},
	))

	strictPolicy := policies.NewStrictPolicy(types.NewRandomPolicy())
	strictPolicy.AddPolicy(policies.If(policies.Always()).Then(types.PickKeepSame()))

	c.AddExperiment(types.NewExperiment(
		"Strict-Policy",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      strictPolicy,
			Environment: getLPaxosPartEnv(lPaxosConfig, true),
		},
	))

	c.Run()
}

func PaxosRewardCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "paxos-reward",
		Run: func(cmd *cobra.Command, args []string) {
			PaxosReward(episodes, horizon, saveFile)
		},
	}
	cmd.PersistentFlags().IntVarP(&requests, "requests", "r", 1, "Number of requests to run with")
	cmd.PersistentFlags().BoolVarP(&timeouts, "timeouts", "t", false, "Run with timeouts or not")
	return cmd
}
