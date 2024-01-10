package types

import "context"

// Environment that RL observes
type Environment interface {
	// Reset called at the end of each episode
	Reset() State
	// Take the corresponding action and return the next state
	Step(Action) State
}

// Ctx methods to propagate the context for timeOut setting
type EnvironmentCtx interface {
	// Reset called at the end of each episode
	ResetCtx(context.Context) (State, error)
	// Step function with context to cancel it
	StepCtx(Action, context.Context) (State, error)
}

// Env implementing both standard and Ctx methods
type EnvironmentUnion interface {
	Environment
	EnvironmentCtx
}

// State of the system that RL policies observe
type State interface {
	// Indexed by the Hash
	// Should be deterministic
	Hash() string
	// Actions possible from the state
	Actions() []Action
}

// And Action that RL policy can take
type Action interface {
	// Index of the action
	// Should be deterministic
	Hash() string
}

type StateAbstractor func(State) string
