package raft

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/zeu5/raft-rl-test/types"
	"go.etcd.io/raft/v3"
	pb "go.etcd.io/raft/v3/raftpb"
)

var _ types.ReplicaState = raft.Status{}

type RaftPartitionColor struct {
	Params map[string]interface{}
}

var _ types.Color = &RaftPartitionColor{}

func (r *RaftPartitionColor) Hash() string {
	bs, _ := json.Marshal(r.Params)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

func (r *RaftPartitionColor) Copy() types.Color {
	new := &RaftPartitionColor{
		Params: make(map[string]interface{}),
	}
	for k, v := range r.Params {
		new.Params[k] = v
	}
	return new
}

type RaftColorFunc func(raft.Status) (string, interface{})

func ColorState() RaftColorFunc {
	return func(s raft.Status) (string, interface{}) {
		return "state", s.RaftState.String()
	}
}

func ColorTerm() RaftColorFunc {
	return func(s raft.Status) (string, interface{}) {
		return "term", s.Term
	}
}

func ColorBoundedTerm(bound int) RaftColorFunc {
	return func(s raft.Status) (string, interface{}) {
		term := s.Term
		if term > uint64(bound) {
			term = uint64(bound)
		}
		return "boundedTerm", term
	}
}

func ColorCommit() RaftColorFunc {
	return func(s raft.Status) (string, interface{}) {
		return "commit", s.Commit
	}
}

func ColorApplied() RaftColorFunc {
	return func(s raft.Status) (string, interface{}) {
		return "applied", s.Applied
	}
}

func ColorVote() RaftColorFunc {
	return func(s raft.Status) (string, interface{}) {
		return "vote", s.Vote
	}
}

func ColorLeader() RaftColorFunc {
	return func(s raft.Status) (string, interface{}) {
		return "leader", s.Lead
	}
}

type RaftStatePainter struct {
	paramFuncs []RaftColorFunc
}

func NewRaftStatePainter(paramFuncs ...RaftColorFunc) *RaftStatePainter {
	return &RaftStatePainter{
		paramFuncs: paramFuncs,
	}
}

func (p *RaftStatePainter) Color(s types.ReplicaState) types.Color {
	rs := s.(raft.Status)
	c := &RaftPartitionColor{
		Params: make(map[string]interface{}),
	}

	for _, p := range p.paramFuncs {
		k, v := p(rs)
		c.Params[k] = v
	}
	return c
}

var _ types.Painter = &RaftStatePainter{}

type RaftMessageWrapper struct {
	pb.Message
}

func (r RaftMessageWrapper) From() uint64 {
	return r.Message.From
}

func (r RaftMessageWrapper) To() uint64 {
	return r.Message.To
}

func (r RaftMessageWrapper) Hash() string {
	return msgKey(r.Message)
}

func (r RaftMessageWrapper) Copy() RaftMessageWrapper {
	newMessage := pb.Message{
		Type:       r.Message.Type,
		To:         r.Message.To,
		From:       r.Message.From,
		Term:       r.Message.Term,
		LogTerm:    r.Message.LogTerm,
		Index:      r.Message.Index,
		Entries:    make([]pb.Entry, len(r.Message.Entries)),
		Commit:     r.Message.Commit,
		Vote:       r.Message.Vote,
		Snapshot:   r.Message.Snapshot,
		Reject:     r.Message.Reject,
		RejectHint: r.Message.RejectHint,
		Context:    r.Message.Context,
		Responses:  r.Message.Responses,
	}
	for i, entry := range r.Message.Entries {
		newMessage.Entries[i] = pb.Entry{
			Term:  entry.Term,
			Index: entry.Index,
			Type:  entry.Type,
			Data:  entry.Data,
		}
	}
	return RaftMessageWrapper{
		Message: newMessage,
	}
}

var _ types.Message = RaftMessageWrapper{}
var _ types.PartitionedSystemState = RaftState{}

var _ types.PartitionedSystemEnvironment = &RaftPartitionEnv{}

type RaftPartitionEnv struct {
	*RaftEnvironment
}

func NewPartitionEnvironment(config RaftEnvironmentConfig) *RaftPartitionEnv {
	return &RaftPartitionEnv{
		RaftEnvironment: NewRaftEnvironment(config),
	}
}

func (p *RaftPartitionEnv) Reset() types.PartitionedSystemState {
	s := p.RaftEnvironment.Reset()
	return s.(RaftState)
}

func (p *RaftPartitionEnv) Tick() types.PartitionedSystemState {
	for _, node := range p.nodes {
		node.Tick()
	}
	newState := RaftState{
		NodeStates:   make(map[uint64]raft.Status),
		WithTimeouts: p.config.Timeouts,
	}
	for id, node := range p.nodes {
		if node.HasReady() {
			ready := node.Ready()
			if !raft.IsEmptySnap(ready.Snapshot) {
				p.storages[id].ApplySnapshot(ready.Snapshot)
			}
			p.storages[id].Append(ready.Entries)
			for _, message := range ready.Messages {
				p.messages[msgKey(message)] = message
			}
			node.Advance(ready)
		}
		newState.NodeStates[id] = node.Status()
	}
	newState.Messages = copyMessages(p.messages)
	p.curState = newState
	return newState
}

func (p *RaftPartitionEnv) DeliverMessage(m types.Message) types.PartitionedSystemState {
	rm := m.(RaftMessageWrapper)
	if rm.Type == pb.MsgProp {
		haveLeader := false
		leader := uint64(0)
		for id, node := range p.nodes {
			if node.Status().RaftState == raft.StateLeader {
				haveLeader = true
				leader = id
				break
			}
		}
		if haveLeader {
			message := rm.Message
			message.To = leader
			p.nodes[leader].Step((message))
			delete(p.messages, msgKey(message))
		}
	} else {
		node := p.nodes[rm.Message.To]
		node.Step(rm.Message)
		delete(p.messages, msgKey(rm.Message))
	}
	newState := RaftState{
		NodeStates:   make(map[uint64]raft.Status),
		WithTimeouts: p.config.Timeouts,
		Logs:         make(map[uint64][]pb.Entry),
	}
	for id, node := range p.nodes {
		if node.HasReady() {
			ready := node.Ready()
			if !raft.IsEmptySnap(ready.Snapshot) {
				p.storages[id].ApplySnapshot(ready.Snapshot)
			}
			p.storages[id].Append(ready.Entries)
			for _, message := range ready.Messages {
				p.messages[msgKey(message)] = message
			}
			node.Advance(ready)
		}
		status := node.Status()
		newState.Logs[id] = make([]pb.Entry, 0)
		ents, err := p.storages[id].Entries(0, status.Commit, status.Commit)
		if err == nil {
			newState.Logs[id] = ents
		}
		newState.NodeStates[id] = status
	}
	newState.Messages = copyMessages(p.messages)
	p.curState = newState
	return newState
}

func (p *RaftPartitionEnv) DropMessage(m types.Message) types.PartitionedSystemState {
	newState := RaftState{
		NodeStates:   copyNodeStates(p.curState.NodeStates),
		Messages:     copyMessages(p.curState.Messages),
		WithTimeouts: p.config.Timeouts,
	}
	delete(newState.Messages, m.Hash())
	p.curState = newState
	return newState
}
