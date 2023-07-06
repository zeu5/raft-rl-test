package lpaxos

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/zeu5/raft-rl-test/types"
)

type LNodeStateColor struct {
	Params map[string]interface{}
}

func (c LNodeStateColor) Hash() string {
	bs, _ := json.Marshal(c.Params)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

func (c LNodeStateColor) Copy() types.Color {
	new := LNodeStateColor{
		Params: make(map[string]interface{}),
	}
	for k, v := range c.Params {
		new.Params[k] = v
	}
	return new
}

type LNodeStatePainter struct {
	paramFuncs []LColorFunc
}

type LColorFunc func(LNodeState) (string, interface{})

func NewLNodeStatePainter(paramfunc ...LColorFunc) *LNodeStatePainter {
	return &LNodeStatePainter{
		paramFuncs: paramfunc,
	}
}

func (l *LNodeStatePainter) Color(s types.ReplicaState) types.Color {
	ls := s.(LNodeState)
	color := LNodeStateColor{
		Params: make(map[string]interface{}),
	}
	for _, pf := range l.paramFuncs {
		key, value := pf(ls)
		color.Params[key] = value
	}
	return color
}

func ColorStep() LColorFunc {
	return func(ls LNodeState) (string, interface{}) {
		return "Step", int(ls.Step)
	}
}

func ColorPhase() LColorFunc {
	return func(ls LNodeState) (string, interface{}) {
		return "Phase", ls.Phase
	}
}

func ColorDecided() LColorFunc {
	return func(ls LNodeState) (string, interface{}) {
		return "Decided", ls.Decided
	}
}

func ColorLeader() LColorFunc {
	return func(ls LNodeState) (string, interface{}) {
		return "Leader", ls.Leader
	}
}

var _ types.Painter = &LNodeStatePainter{}
var _ types.Message = &Message{}
var _ types.ReplicaState = LNodeState{}
var _ types.PartitionedSystemState = &LPaxosState{}

type LPaxosPartitionEnv struct {
	*LPaxosEnv
}

var _ types.PartitionedSystemEnvironment = &LPaxosPartitionEnv{}

func NewLPaxosPartitionEnv(c LPaxosEnvConfig) *LPaxosPartitionEnv {
	return &LPaxosPartitionEnv{
		LPaxosEnv: NewLPaxosEnv(c),
	}
}

func (l *LPaxosPartitionEnv) Reset() types.PartitionedSystemState {
	s := l.LPaxosEnv.Reset()
	return s.(*LPaxosState)
}

func (l *LPaxosPartitionEnv) Tick() types.PartitionedSystemState {
	for _, node := range l.nodes {
		node.Tick()
	}
	newState := &LPaxosState{
		NodeStates:   make(map[uint64]LNodeState),
		WithTimeouts: l.config.Timeouts,
	}
	for id, node := range l.nodes {
		rd := node.Ready()
		for _, m := range rd.Messages {
			l.messages[m.Hash()] = m
		}
		newState.NodeStates[id] = node.Status()
	}
	newState.Messages = copyMessages(l.messages)
	l.curState = newState
	return newState
}

func (l *LPaxosPartitionEnv) DeliverMessage(m types.Message) types.PartitionedSystemState {
	lm := m.(Message)
	if lm.Type == CommandMessage {
		haveLeader := false
		var leader uint64 = 0
		for id, n := range l.curState.NodeStates {
			if n.Leader == id {
				leader = id
				haveLeader = true
				break
			}
		}
		if haveLeader {
			message := lm
			message.T = leader
			l.nodes[leader].Step(message)
			delete(l.messages, message.Hash())
		}
	} else {
		message := lm
		l.nodes[message.T].Step(message)
		delete(l.messages, message.Hash())
	}
	newState := &LPaxosState{
		NodeStates:   make(map[uint64]LNodeState),
		WithTimeouts: l.config.Timeouts,
	}
	for id, node := range l.nodes {
		rd := node.Ready()
		for _, m := range rd.Messages {
			l.messages[m.Hash()] = m
		}
		newState.NodeStates[id] = node.Status()
	}
	newState.Messages = copyMessages(l.messages)
	l.curState = newState
	return newState
}

func (l *LPaxosPartitionEnv) DropMessage(m types.Message) types.PartitionedSystemState {
	newState := &LPaxosState{
		NodeStates:   copyNodeStates(l.curState.NodeStates),
		Messages:     copyMessages(l.curState.Messages),
		WithTimeouts: l.config.Timeouts,
	}
	delete(newState.Messages, m.Hash())
	l.curState = newState
	return newState
}
