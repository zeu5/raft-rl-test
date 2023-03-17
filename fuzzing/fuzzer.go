package fuzzing

import (
	"math/rand"
	"time"

	pb "go.etcd.io/raft/v3/raftpb"
)

type Fuzzer struct {
	MessageQueues               map[string]*Queue[pb.Message]
	Nodes                       []string
	Config                      *FuzzerConfig
	Mutator                     Mutator
	MutatedNodeChoices          *Queue[string]
	CurEventTrace               EventTrace
	CurTrace                    Trace
	MutatedRandomBooleanChoices *Queue[bool]
	MutatedRandomIntegerChoices *Queue[int]
	rand                        *rand.Rand
}

type FuzzerConfig struct {
	Iterations int
	Steps      int
	Mutator    Mutator
}

func NewFuzzer(config *FuzzerConfig) *Fuzzer {
	return &Fuzzer{
		Config:                      config,
		Nodes:                       make([]string, 0),
		MessageQueues:               make(map[string]*Queue[pb.Message]),
		Mutator:                     config.Mutator,
		MutatedNodeChoices:          NewQueue[string](),
		CurEventTrace:               NewEventTrace(),
		CurTrace:                    NewTrace(),
		MutatedRandomBooleanChoices: NewQueue[bool](),
		MutatedRandomIntegerChoices: NewQueue[int](),
		rand:                        rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (f *Fuzzer) GetRandomBoolean() (choice bool) {
	if f.MutatedRandomBooleanChoices.Size() > 0 {
		choice, _ = f.MutatedRandomBooleanChoices.Pop()
	} else {
		choice = f.rand.Intn(2) == 0
	}
	f.CurEventTrace = append(f.CurEventTrace, &Event{
		Name: "RandomBooleanChoice",
		Params: map[string]interface{}{
			"choice": choice,
		},
	})
	f.CurTrace = append(f.CurTrace, &SchedulingChoice{
		Type:          RandomBoolean,
		BooleanChoice: choice,
	})
	return
}

func (f *Fuzzer) GetRandomInteger(max int) (choice int) {
	if f.MutatedRandomIntegerChoices.Size() > 0 {
		choice, _ = f.MutatedRandomIntegerChoices.Pop()
	} else {
		choice = f.rand.Intn(max)
	}
	f.CurEventTrace = append(f.CurEventTrace, &Event{
		Name: "RandomIntegerChoice",
		Params: map[string]interface{}{
			"choice": choice,
		},
	})
	f.CurTrace = append(f.CurTrace, &SchedulingChoice{
		Type:          RandomInteger,
		IntegerChoice: choice,
	})
	return
}

func (f *Fuzzer) GetNextMessage() (message pb.Message, ok bool) {
	var nextNode string
	if f.MutatedNodeChoices.Size() > 0 {
		nextNode, _ = f.MutatedNodeChoices.Pop()
	} else {
		randIndex := f.rand.Intn(len(f.Nodes))
		nextNode = f.Nodes[randIndex]
	}
	message, ok = f.MessageQueues[nextNode].Pop()
	f.CurEventTrace = append(f.CurEventTrace, &Event{
		Name:   message.Type.String(),
		Params: map[string]interface{}{},
	})
	f.CurTrace = append(f.CurTrace, &SchedulingChoice{
		Type:   Node,
		NodeID: nextNode,
	})
	return
}

func (f *Fuzzer) Run() error {
	return nil
}

type Mutator interface {
	Mutate(*Trace) (*Trace, bool)
}

type FuzzContext struct {
	fuzzer *Fuzzer
}

func (f *FuzzContext) AddEvent(e *Event) {
	f.fuzzer.CurEventTrace = append(f.fuzzer.CurEventTrace, e)
}

func (f *FuzzContext) RandomBooleanChoice() bool {
	return f.fuzzer.GetRandomBoolean()
}

func (f *FuzzContext) RandomIntegerChoice(max int) int {
	return f.fuzzer.GetRandomInteger(max)
}
