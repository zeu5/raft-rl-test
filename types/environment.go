package types

// Ctx methods to propagate the context for timeOut setting
type Environment interface {
	// Reset called at the end of each episode
	Reset(*EpisodeContext) (State, error)
	// Step function with context to cancel it
	Step(Action, *StepContext) (State, error)
	// Set the underlying env based on the parallel index
	SetUp(int)
}

// State of the system that RL policies observe
type State interface {
	// Indexed by the Hash
	// Should be deterministic
	Hash() string
	// Actions possible from the state
	Actions() []Action
	// if the state is terminal
	Terminal() bool
}

// And Action that RL policy can take
type Action interface {
	// Index of the action
	// Should be deterministic
	Hash() string
}

type StateAbstractor func(State) string
