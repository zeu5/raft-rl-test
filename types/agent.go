package types

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type AgentConfig struct {
	Episodes    int
	Horizon     int
	Policy      Policy
	Environment EnvironmentUnion
}

// RL Agent configured with the corresponding
// policy and environment
type Agent struct {
	config *AgentConfig
	// collects the traces of the run
	// Only populated if the Run function is invoked
	traces      []*Trace
	policy      Policy
	environment EnvironmentUnion
}

// Instantiates a new Agent
func NewAgent(config *AgentConfig) *Agent {
	return &Agent{
		config: config,
		// TODO: Initialize traces to 0 len
		traces:      make([]*Trace, 0),
		policy:      config.Policy,
		environment: config.Environment,
	}
}

// Run the agent for the specified number of episodes and horizon
func (a *Agent) Run() {
	for i := 0; i < a.config.Episodes; i++ {
		a.traces = append(a.traces, a.runEpisode(i, context.TODO()))
	}
}

func (a *Agent) RunWithCtx(ctx context.Context) {
	for i := 0; i < a.config.Episodes; i++ {
		a.traces = append(a.traces, a.runEpisode(i, ctx))
	}
}

// run a single episode and return the resulting trace
func (a *Agent) runEpisode(episode int, ctx context.Context) *Trace {
	state := a.environment.Reset()
	trace := NewTrace()
	actions := state.Actions()

	// stepTimes := make(map[string][]time.Duration, 0)

EPISODE_LOOP:
	for i := 0; i < a.config.Horizon; i++ {
		select {
		case <-ctx.Done():
			fmt.Println("Stopping!")
			// TODO: Maybe Reset?
			break EPISODE_LOOP
		default:
		}
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

// run a single episode and return the resulting trace
func (a *Agent) runEpisodeWithTimeout(episode int, ctx context.Context, epCtx *EpisodeContext) (*Trace, error) {
	var start time.Time
	var duration time.Duration

	select {
	case <-epCtx.TimeoutContext.Done():
		return nil, errors.New("episode timed out")
	default:

	}

	start = time.Now()
	state, err := a.environment.ResetCtx(epCtx)
	duration = time.Since(start)

	if err != nil {
		return nil, err
	}

	epCtx.Report.AddTimeEntry(duration, "reset_time", "agent.runEpisodeWithTimeout")

	trace := NewTrace()
	actions := state.Actions()

	// stepTimes := make(map[string][]time.Duration, 0)

	// EPISODE_LOOP:
	for i := 0; i < a.config.Horizon; i++ {
		select {
		case <-ctx.Done():
			fmt.Println("Stopping!")
			return nil, errors.New("stopping")
			// break EPISODE_LOOP
		case <-epCtx.TimeoutContext.Done():
			return nil, errors.New("runEpisodeWithTimeout : episode timed out")
		default:
		}

		epCtx.Report.setEpisodeStep(i) // set the current episode step in the report

		if len(actions) == 0 {
			break
		}

		start = time.Now()
		nextAction, ok := a.policy.NextAction(i, state, actions)
		duration = time.Since(start)
		epCtx.Report.AddTimeEntry(duration, "next_action_time", "agent.runEpisodeWithTimeout")

		if !ok {
			break
		}

		start = time.Now()
		nextState, err := a.environment.StepCtx(nextAction, epCtx)
		duration = time.Since(start)
		epCtx.Report.AddTimeEntry(duration, "step_time", "agent.runEpisodeWithTimeout")

		if err != nil {
			return nil, err
		}

		select {
		case <-epCtx.TimeoutContext.Done():
			return nil, errors.New("runEpisodeWithTimeout : episode timed out")
		default:

		}
		a.policy.Update(i, state, nextAction, nextState)

		trace.Append(i, state, nextAction, nextState)
		state = nextState
		actions = nextState.Actions()
	}

	select {
	case <-ctx.Done():
		fmt.Println("Stopping!")
		return nil, errors.New("stopping")
	case <-epCtx.TimeoutContext.Done():
		return nil, errors.New("runEpisodeWithTimeout : episode timed out")
	default:
		a.policy.UpdateIteration(episode, trace)
		return trace, nil
	}
	// for name, times := range stepTimes {
	// 	sum := time.Duration(0)
	// 	for _, d := range times {
	// 		sum += d
	// 	}
	// 	avg := time.Duration(int(sum) / len(times))
	// 	fmt.Printf("Average step time for action:%s, %s\n", name, avg.String())
	// }
}
