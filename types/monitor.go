package types

var (
	InitState string = "init"
	FailState string = "fail"
)

type RewardFunc func(State) bool

type MonitorState struct {
	Success     bool
	Name        string
	transitions map[string]MonitorCondition
}

type MonitorCondition func(State, Action, State) bool

func (m MonitorCondition) Not() MonitorCondition {
	return func(s State, a Action, ns State) bool {
		return !m(s, a, ns)
	}
}

func (m MonitorCondition) Or(other MonitorCondition) MonitorCondition {
	return func(s State, a Action, ns State) bool {
		return m(s, a, ns) || other(s, a, ns)
	}
}

func (m MonitorCondition) And(other MonitorCondition) MonitorCondition {
	return func(s State, a Action, ns State) bool {
		return m(s, a, ns) && other(s, a, ns)
	}
}

type Monitor struct {
	states map[string]*MonitorState
}

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

func (m *Monitor) Build() *MonitorBuilder {
	return &MonitorBuilder{
		monitor:  m,
		curState: m.states[InitState],
	}
}

type MonitorBuilder struct {
	monitor  *Monitor
	curState *MonitorState
}

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

func (m *MonitorBuilder) MarkSuccess() *MonitorBuilder {
	m.curState.Success = true
	return m
}
