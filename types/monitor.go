package types

var (
	InitState string = "init"
	FailState string = "fail"
)

type MonitorState struct {
	Success     bool
	Name        string
	transitions map[string]MonitorCondition
}

type MonitorCondition func(trace *Trace) bool

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
	for i := 1; i <= t.Len(); i++ {
		prefix, _ := t.GetPrefix(i)
		transitioned := false
		for next, cond := range curState.transitions {
			if cond(prefix) {
				transitioned = true
				curState = m.states[next]
			}
		}
		if !transitioned {
			// The case when there is no transition defined
			return nil, false
		}
		if curState.Success {
			return prefix, true
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
