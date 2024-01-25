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
	state, err := a.environment.Reset(ctx)
	duration := time.Since(start)
	if err != nil {
		ctx.SetError(err)
		return
	}
	ctx.Report.AddTimeEntry(duration, "reset_time", "agent.RunEpisode")

	actions := state.Actions()

	for i := 0; i < a.config.Horizon; i++ {
		select {
		case <-ctx.Context.Done():
			return
		default:
		}
		ctx.SetStep(i)
		sCtx := NewStepContext(ctx)
		sCtx.State = state
		if len(actions) == 0 {
			break
		}
		start = time.Now()
		nextAction, ok := a.policy.NextAction(i, state, actions)
		duration = time.Since(start)
		ctx.Report.AddTimeEntry(duration, "next_action_time", "agent.RunEpisode")
		if !ok {
			break
		}
		sCtx.Action = nextAction
		start = time.Now()
		nextState, err := a.environment.Step(nextAction, sCtx)
		duration = time.Since(start)
		ctx.Report.AddTimeEntry(duration, "step_time", "agent.RunEpisode")
		if err != nil {
			ctx.SetError(err)
			return
		}
		sCtx.NextState = nextState
		select {
		case <-ctx.Context.Done():
			return
		default:
		}

		a.policy.Update(sCtx)

		ctx.Trace.AppendCtx(sCtx)
		state = nextState
		actions = nextState.Actions()
	}
	select {
	case <-ctx.Context.Done():
		return
	default:
		a.policy.UpdateIteration(ctx.Episode, ctx.Trace)
	}
}
