package types

import (
	"time"
)

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
	policy      Policy
	environment Environment
}

// Instantiates a new Agent
func NewAgent(config *AgentConfig) *Agent {
	return &Agent{
		config: config,
		// TODO: Initialize traces to 0 len
		policy:      config.Policy,
		environment: config.Environment,
	}
}

func (a *Agent) RunEpisode(ctx *EpisodeContext) {

	select {
	case <-ctx.Context.Done():
		return
	default:
	}

	start := time.Now()
	state, err := a.environment.Reset(ctx) // reset the environment
	duration := time.Since(start)
	if err != nil {
		ctx.SetError(err)
		return
	}
	ctx.Report.AddTimeEntry(duration, "reset_time", "agent.RunEpisode")

	actions := state.Actions() // get the available actions

EpLoop:
	for i := 0; i < a.config.Horizon; i++ {
		select {
		case <-ctx.Context.Done():
			ctx.SetTimedOut()
			break EpLoop
		default:
		}
		ctx.SetStep(i)
		sCtx := NewStepContext(ctx) // create a new step context, stores STATE, ACTION, NEXT_STATE plus additional information
		sCtx.State = state          // store the current state in the step context
		if len(actions) == 0 {
			break
		}
		start = time.Now()
		nextAction, ok := a.policy.NextAction(i, state, actions) // query the policy for the next action
		duration = time.Since(start)
		ctx.Report.AddTimeEntry(duration, "next_action_time", "agent.RunEpisode")
		if !ok {
			break
		}
		sCtx.Action = nextAction // store the action in the step context

		start = time.Now()
		nextState, err := a.environment.Step(nextAction, sCtx) // call the environment step function
		duration = time.Since(start)
		ctx.Report.AddTimeEntry(duration, "step_time", "agent.RunEpisode")
		if err != nil {
			ctx.SetError(err)
			break
		}
		sCtx.NextState = nextState // store the next state in the step context
		select {
		case <-ctx.Context.Done():
			ctx.SetTimedOut()
			break EpLoop
		default:
		}

		// single step update, usually not used
		a.policy.Update(sCtx)

		ctx.Trace.AppendCtx(sCtx) // append the step context to the trace
		state = nextState

		if state.Terminal() { // terminate if the state is terminal
			break
		}
		actions = nextState.Actions()
	}
	// set the number of timesteps taken in the episode
	ctx.Timesteps = ctx.Trace.Len()
	select {
	case <-ctx.Context.Done():
		ctx.SetTimedOut()
		return // skip policy update if the episode was canceled (error or timeout)
	default:
		// multi-step backward update, if the episode did not have error or timeout
		if ctx.Err == nil && !ctx.TimedOut {
			a.policy.UpdateIteration(ctx.Episode, ctx.Trace)
		}
	}
}
