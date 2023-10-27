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

	// define predicates
	decided := lpaxos.Decided()
	inPhase := lpaxos.InPhase(3)
	// inStep := lpaxos.InStep(lpaxos.StepPromise)
	onlyMajorityDecided := lpaxos.OnlyMajorityDecidedOne()
	emptyLogLeader := lpaxos.EmptyLogLeader().And(lpaxos.InPhase(4)).And(decided)

	// build a machine with sequence of predicates
	checkRM := policies.NewRewardMachine(emptyLogLeader)           // final predicate - target space
	checkRM.AddState(onlyMajorityDecided, "OnlyMajorityDecided")   // 1st step
	checkRM.AddState(inPhase.And(onlyMajorityDecided), "InPhase3") // 2nd ...
	checkRM.AddState(lpaxos.InPhase(4).And(decided), "Decided")

	guideRM := policies.NewRewardMachine(onlyMajorityDecided)
	// guideRM.AddState(onlyMajorityDecided, "OnlyMajorityDecided")

	c := types.NewComparison(types.BugAnalyzer(types.BugDesc{Name: "Safety", Check: lpaxos.SafetyBug()}), types.BugComparator(saveFile), runs)
	// c := types.NewComparison(policies.PredicatesAnalyzer(onlyMajorityDecided, inPhase, emptyLogLeader), policies.PredicatesComparator())
	// c := types.NewComparison(lpaxos.LPaxosAnalyzer(saveFile), lpaxos.LPaxosComparator(saveFile))
	// c := types.NewComparison(policies.RewardMachineAnalyzer(checkRM), policies.RewardMachineCoverageComparator(saveFile), runs)
	c.AddExperiment(types.NewExperiment(
		"Random-Part",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      types.NewRandomPolicy(),
			Environment: getLPaxosPartEnv(lPaxosConfig, true),
		},
	))

	strictPolicy := policies.NewStrictPolicy(types.NewRandomPolicy())
	strictPolicy.AddPolicy(policies.If(policies.Always()).Then(types.PickKeepSame()))
	c.AddExperiment(types.NewExperiment(
		"Strict",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      strictPolicy,
			Environment: getLPaxosPartEnv(lPaxosConfig, true),
		},
	))

	c.AddExperiment(types.NewExperiment(
		"Exploration",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      policies.NewBonusPolicyGreedy(0.1, 0.99, 0.2),
			Environment: getLPaxosPartEnv(lPaxosConfig, true),
		},
	))
	c.AddExperiment(types.NewExperiment(
		"RewardMachine",
		&types.AgentConfig{
			Episodes:    episodes,
			Horizon:     horizon,
			Policy:      policies.NewRewardMachinePolicy(guideRM),
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
