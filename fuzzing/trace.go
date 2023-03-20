package fuzzing

type Event struct {
	Name   string
	Params map[string]interface{}
	Reset  bool
}

type EventTrace []*Event

func NewEventTrace() EventTrace {
	return EventTrace(make([]*Event, 0))
}

var (
	Node          SchedulingChoiceType = "Node"
	RandomBoolean SchedulingChoiceType = "RandomBoolean"
	RandomInteger SchedulingChoiceType = "RandomInteger"
)

type SchedulingChoiceType string

type SchedulingChoice struct {
	Type          SchedulingChoiceType
	NodeID        uint64
	BooleanChoice bool
	IntegerChoice int
}

type Trace []*SchedulingChoice

func NewTrace() Trace {
	return Trace(make(Trace, 0))
}
