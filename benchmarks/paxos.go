package benchmarks

import (
	"context"
	"time"

	"github.com/zeu5/raft-rl-test/lpaxos"
	"github.com/zeu5/raft-rl-test/policies"
	"github.com/zeu5/raft-rl-test/types"
)

func Paxos(episodes, horizon, runs int, saveFile string, ctx context.Context) {
	// The configuration for the paxos environment
	lPaxosConfig := lpaxos.LPaxosEnvConfig{
		// Number of replicas to run
		Replicas: 3,
		// Number of initial requests to inject into the system
		Requests: requests,
		// The timeout value in terms of number of ticks
		Timeout: 12,
		// Timeouts is a boolean flag to indicate if the environment has drop message actions or not
		Timeouts: timeouts,
	}

	// property := lpaxos.InconsistentLogs()
	// Comparison runs different agents as specified below. Then analyzes the traces for each agent configuration and compares them
	c := types.NewComparison(&types.ComparisonConfig{
		Runs:       runs,
		Episodes:   episodes,
		Horizon:    horizon,
		RecordPath: saveFile,
		Timeout:    0 * time.Second,
		// record flags
		RecordTraces: false,
		RecordTimes:  false,
		RecordPolicy: false,
		// last traces
		PrintLastTraces:     0,
		PrintLastTracesFunc: nil,
		// report config
		ReportConfig: types.RepConfigOff(),
	})
	c.AddAnalysis("Plot", lpaxos.NewLPaxosAnalyzer(saveFile), lpaxos.LPaxosComparator(saveFile))
	// Adding the different policy and experiments
	c.AddExperiment(types.NewExperiment(
		"RL",
		types.NewSoftMaxNegPolicy(0.3, 0.7, 1),
		getLPaxosEnv(lPaxosConfig, abstracter),
	))
	c.AddExperiment(types.NewExperiment(
		"Random",
		types.NewRandomPolicy(),
		getLPaxosEnv(lPaxosConfig, abstracter),
	))
	c.AddExperiment(types.NewExperiment(
		"BonusMaxRL",
		policies.NewBonusPolicyGreedy(0.1, 0.99, 0.2),
		getLPaxosEnv(lPaxosConfig, abstracter),
	))
	c.AddExperiment(types.NewExperiment(
		"BonusSoftMaxRL",
		policies.NewBonusPolicySoftMax(0.1, 0.99, 0.01),
		getLPaxosEnv(lPaxosConfig, abstracter),
	))

	// Invoking the different experiments
	c.Run(ctx)
}

func getLPaxosEnv(config lpaxos.LPaxosEnvConfig, abs string) types.Environment {
	if abs == "none" {
		return lpaxos.NewLPaxosEnv(config)
	}
	abstracter := lpaxos.DefaultAbstractor()
	switch abs {
	case "ignore-phase":
		abstracter = lpaxos.IgnorePhase()
	case "ignore-last":
		abstracter = lpaxos.IgnoreLast()
	}
	return lpaxos.NewLPaxosAbsEnv(config, abstracter)
}
