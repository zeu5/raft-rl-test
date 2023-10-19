package main

import (
	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/rsl"
	"github.com/zeu5/raft-rl-test/types"
)

func RSLRewardMachine() {
	config := rsl.RSLEnvConfig{
		Nodes: 3,
		NodeConfig: rsl.NodeConfig{
			HeartBeatInterval:       2,
			NoProgressTimeout:       15,
			BaseElectionDelay:       10,
			InitializeRetryInterval: 5,
			NewLeaderGracePeriod:    15,
			VoteRetryInterval:       5,
			PrepareRetryInterval:    5,
			MaxCachedLength:         10,
			ProposalRetryInterval:   5,
		},
		NumCommands:        requests,
		AdditionalCommands: make([]rsl.Command, 0),
	}

	m1 := policies.NewRewardMachine(rsl.NodePrimary(1))

	// m2 := policies.NewRewardMachine(rsl.NodePrimary(2))
	// m2.AddState(rsl.NodePrimary(1), "OnePrimary")

	// m3 := policies.NewRewardMachine(rsl.NodeNumDecided(1, 2).And(rsl.NodePrimary(1)))

	// m4 := policies.NewRewardMachine(rsl.NodeNumDecided(1, 2))
	// m4.AddState(rsl.NodePrimary(1).And(rsl.NumDecided(0)), "NodeOnePrimary")

	// m5 := policies.NewRewardMachine(rsl.InBallot(2))

	// m6 := policies.NewRewardMachine(rsl.NodeInBallot(1,2))

	// m7 := policies.NewRewardMachine(rsl.InPreparedBallot(2))

	// m8 := policies.NewRewardMachine(rsl.NodeInPreparedBallot(1, 2))

	guideRM := policies.NewRewardMachine(rsl.InState(rsl.StateStablePrimary))
	// monitorRM := policies.NewRewardMachine(rsl.Decided())

	c := types.NewComparison(policies.RewardMachineAnalyzer(m1), policies.RewardMachineCoverageComparator(), runs)
	c.AddExperiment(types.NewExperiment(
		"random",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      types.NewRandomPolicy(),
			Environment: GetRSLEnvironment(config),
		},
	))
	strictPolicy := policies.NewStrictPolicy(types.NewRandomPolicy())
	strictPolicy.AddPolicy(policies.If(policies.Always()).Then(types.PickKeepSame()))

	c.AddExperiment(types.NewExperiment(
		"Strict",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      policies.NewStrictPolicy(strictPolicy),
			Environment: GetRSLEnvironment(config),
		},
	))
	c.AddExperiment(types.NewExperiment(
		"BonusMax",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      policies.NewBonusPolicyGreedy(0.1, 0.99, 0.2),
			Environment: GetRSLEnvironment(config),
		},
	))
	c.AddExperiment(types.NewExperiment(
		"RewardMachine",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      policies.NewRewardMachinePolicy(guideRM),
			Environment: GetRSLEnvironment(config),
		},
	))

	c.Run()
}

func RSLRewardMachineCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "rsl-rm",
		Run: func(cmd *cobra.Command, args []string) {
			RSLRewardMachine()
		},
	}
	cmd.PersistentFlags().IntVarP(&requests, "requests", "r", 1, "Number of requests to run with")
	return cmd
}
