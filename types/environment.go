package types

// Ctx methods to propagate the context for timeOut setting
type Environment interface {
	// Reset called at the end of each episode
	Reset(*EpisodeContext) (State, error)
	// Step function with context to cancel it
	Step(Action, *StepContext) (State, error)
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
