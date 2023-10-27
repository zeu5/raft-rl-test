package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/rsl"
	"github.com/zeu5/raft-rl-test/types"
)

func getRewardMachine(name string) (*policies.RewardMachine, bool) {
	var machine *policies.RewardMachine = nil
	switch name {
	case "InStatePrimary":
		machine = policies.NewRewardMachine(rsl.InState(rsl.StateStablePrimary))
	case "SinglePrimary":
		machine = policies.NewRewardMachine(rsl.NodePrimary(1))
	case "NumDecided":
		machine = policies.NewRewardMachine(rsl.NumDecided(2))
	case "ChangePrimary":
		m2 := policies.NewRewardMachine(rsl.NodePrimary(2))
		m2.AddState(rsl.NodePrimary(1), "OnePrimary")
		machine = m2
	case "NodeDecidedAndPrimary":
		machine = policies.NewRewardMachine(rsl.NodeNumDecided(1, 2).And(rsl.NodePrimary(1)))
	case "NodeDecidedAfterPrimary":
		m4 := policies.NewRewardMachine(rsl.NodeNumDecided(1, 2))
		m4.AddState(rsl.NodePrimary(1).And(rsl.NumDecided(0)), "NodeOnePrimary")
		machine = m4
	case "InBallot":
		machine = policies.NewRewardMachine(rsl.InBallot(2))
	case "NodeInBallot":
		machine = policies.NewRewardMachine(rsl.NodeInBallot(1, 2))
	case "InPreparedBallot":
		machine = policies.NewRewardMachine(rsl.InPreparedBallot(2))
	case "NodeInPreparedBallot":
		machine = policies.NewRewardMachine(rsl.NodeInPreparedBallot(1, 2))
	}
	return machine, machine != nil
}

func RSLRewardMachine(rewardMachine string) error {
	if rewardMachine == "" {
		return errors.New("please specify a reward machine")
	}
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

	rm, ok := getRewardMachine(rewardMachine)
	if !ok {
		return fmt.Errorf("cannot find reward machine: %s", rewardMachine)
	}

	colors := []rsl.RSLColorFunc{rsl.ColorState(), rsl.ColorDecree(), rsl.ColorDecided(), rsl.ColorBoundedBallot(5)}

	c := types.NewComparison(policies.RewardMachineAnalyzer(rm), policies.RewardMachineCoverageComparator(saveFile), runs)
	c.AddExperiment(types.NewExperiment(
		"random",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      types.NewRandomPolicy(),
			Environment: GetRSLEnvironment(config, colors),
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
			Environment: GetRSLEnvironment(config, colors),
		},
	))
	c.AddExperiment(types.NewExperiment(
		"BonusMax",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      policies.NewBonusPolicyGreedy(0.1, 0.99, 0.2),
			Environment: GetRSLEnvironment(config, colors),
		},
	))
	c.AddExperiment(types.NewExperiment(
		"RewardMachine",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      policies.NewRewardMachinePolicy(rm),
			Environment: GetRSLEnvironment(config, colors),
		},
	))

	c.Run()
	return nil
}

func RSLRewardMachineCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "rsl-rm reward_machine",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RSLRewardMachine(args[0])
		},
	}
	cmd.PersistentFlags().IntVarP(&requests, "requests", "r", 1, "Number of requests to run with")
	return cmd
}
