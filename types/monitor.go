package types

var (
	InitState string = "init"
	FailState string = "fail"
)

// Primitive reward function to determine which states to give a reward to
// Currently defined on a transition (pair of states)
type RewardFunc func(State, State) bool

// Primitive reward function to determine which states to give a reward to
// Currently defined on a transition (pair of states)
type RewardFuncSingle func(State) bool

func (r RewardFuncSingle) And(other RewardFuncSingle) RewardFuncSingle {
	return func(s State) bool {
		return r(s) && other(s)
	}
}

func (r RewardFuncSingle) Or(other RewardFuncSingle) RewardFuncSingle {
	return func(s State) bool {
		return r(s) || other(s)
	}
}

func (r RewardFuncSingle) Not() RewardFuncSingle {
	return func(s State) bool {
		return !r(s)
	}
}

// MonitorState is a state in the state machine (Monitor)
// Use MonitorBuilder to create monitor states (do not instantiate directly)
type MonitorState struct {
	Success     bool
	Name        string
	transitions map[string]MonitorCondition
}

// Transitions of a Monitor are labelled with a MonitorCondition
// MonitorCondition is a predicate on the transition of RL (state, action, nextState)
type MonitorCondition func(State, Action, State) bool

// Not operator on the MonitorCondition
func (m MonitorCondition) Not() MonitorCondition {
	return func(s State, a Action, ns State) bool {
		return !m(s, a, ns)
	}
}

// Or operator between MonitorCondition's
func (m MonitorCondition) Or(other MonitorCondition) MonitorCondition {
	return func(s State, a Action, ns State) bool {
		return m(s, a, ns) || other(s, a, ns)
	}
}

// And operator between MonitorCondition's
func (m MonitorCondition) And(other MonitorCondition) MonitorCondition {
	return func(s State, a Action, ns State) bool {
		return m(s, a, ns) && other(s, a, ns)
	}
}

// Monitor is a generic state machine
// Could be used to crate reward machines
type Monitor struct {
	states map[string]*MonitorState
}

// Checks if a trace satisfies the monitor
// Simulates the monitor and returns the prefix
// that results in a transition to a success state
func (m *Monitor) Check(t *Trace) (*Trace, bool) {
	satisfyingPrefix := NewTrace()
	curState := m.states[InitState]
	if t.Len() == 0 || curState.Success {
		// The case when the initial state is successful
		// Or there are no steps in the trace and the initial state is successful
		return satisfyingPrefix, curState.Success
	}
	for i := 0; i < t.Len(); i++ {
		s, a, ns, _ := t.Get(i)
		// transitioned := false
		for next, cond := range curState.transitions {
			if cond(s, a, ns) {
				// transitioned = true
				curState = m.states[next]
				break
			}
		}
		// if !transitioned {
		// 	// The case when there is no transition defined
		// 	return nil, false
		// }
		if curState.Success {
			return t.GetPrefix(i)
		}
	}
	return nil, false
}

// Creates a new Monitor
// with a default initial state
func NewMonitor() *Monitor {
	m := &Monitor{
		states: make(map[string]*MonitorState),
	}
	m.states[InitState] = &MonitorState{
		Name:        InitState,
		Success:     false,
		transitions: make(map[string]MonitorCondition),
	}
	return m
}

// Returns a MonitorBuilder to construct the remainder of the state machine
// Initialized at the initial state
func (m *Monitor) Build() *MonitorBuilder {
	return &MonitorBuilder{
		monitor:  m,
		curState: m.states[InitState],
	}
}

// Encodes a Builder pattern to create the state machine
// The builder is indexed at a particular state of the state machine (Monitor)
type MonitorBuilder struct {
	monitor  *Monitor
	curState *MonitorState
}

// On defines a transition from the current state based on the condition the next state
// returns a new builder instance that is indexed at the next state.
// To construct a chain of state one can call s1.On().On().On()...
// Node: If `next` is not part of the state machine, then its newly created otherwise the existing state is indexed
func (m *MonitorBuilder) On(cond MonitorCondition, next string) *MonitorBuilder {
	var nextState *MonitorState = nil
	if _, ok := m.monitor.states[next]; !ok {
		nextState = &MonitorState{
			Name:        next,
			Success:     false,
			transitions: make(map[string]MonitorCondition),
		}
		m.monitor.states[next] = nextState
	} else {
		nextState = m.monitor.states[next]
	}
	m.curState.transitions[next] = cond
	return &MonitorBuilder{
		monitor:  m.monitor,
		curState: nextState,
	}
}

// Mark the corresponding state indexed at this builder instance as a success state
func (m *MonitorBuilder) MarkSuccess() *MonitorBuilder {
	m.curState.Success = true
	return m
}
