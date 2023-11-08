package types

type AgentConfig struct {
	Episodes    int
	Horizon     int
	Policy      Policy
	Environment Environment
}

// RL Agent configured with the corresponding
// policy and environment
type Agent struct {
	config *AgentConfig
	// collects the traces of the run
	// Only populated if the Run function is invoked
	traces      []*Trace
	policy      Policy
	environment Environment
}

// Instantiates a new Agent
func NewAgent(config *AgentConfig) *Agent {
	return &Agent{
		config:      config,
		traces:      make([]*Trace, config.Episodes),
		policy:      config.Policy,
		environment: config.Environment,
	}
}

// Run the agent for the specified number of episodes and horizon
func (a *Agent) Run() {
	for i := 0; i < a.config.Episodes; i++ {
		a.traces[i] = a.runEpisode(i)
	}
}

// run a single episode and return the resulting trace
func (a *Agent) runEpisode(episode int) *Trace {
	state := a.environment.Reset()
	trace := NewTrace()
	actions := state.Actions()

	// stepTimes := make(map[string][]time.Duration, 0)

	for i := 0; i < a.config.Horizon; i++ {
		if len(actions) == 0 {
			break
		}
		nextAction, ok := a.policy.NextAction(i, state, actions)
		if !ok {
			break
		}
		// start := time.Now()
		// actionName := reflect.TypeOf(nextAction).Elem().Name()
		nextState := a.environment.Step(nextAction)
		// dur := time.Since(start)
		// if _, ok := stepTimes[actionName]; !ok {
		// 	stepTimes[actionName] = make([]time.Duration, 0)
		// }
		// stepTimes[actionName] = append(stepTimes[actionName], dur)
		a.policy.Update(i, state, nextAction, nextState)

		trace.Append(i, state, nextAction, nextState)
		state = nextState
		actions = nextState.Actions()
	}
	a.policy.UpdateIteration(episode, trace)

	// for name, times := range stepTimes {
	// 	sum := time.Duration(0)
	// 	for _, d := range times {
	// 		sum += d
	// 	}
	// 	avg := time.Duration(int(sum) / len(times))
	// 	fmt.Printf("Average step time for action:%s, %s\n", name, avg.String())
	// }

	return trace
}
