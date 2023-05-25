package raft

import (
	"github.com/zeu5/raft-rl-test/types"
	"go.etcd.io/raft/v3"
	pb "go.etcd.io/raft/v3/raftpb"
)

var _ types.ReplicaState = raft.Status{}

type RaftStatePainter struct {
}

func (p *RaftStatePainter) Color(s types.ReplicaState) types.Color {
	rs := s.(raft.Status)
	return types.Color(rs.RaftState)
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
	abs StateAbstracter
}

func NewPartitionEnvironment(config RaftEnvironmentConfig, abstracter StateAbstracter) *RaftPartitionEnv {
	return &RaftPartitionEnv{
		RaftEnvironment: NewRaftEnvironment(config),
		abs:             abstracter,
	}
}

func (p *RaftPartitionEnv) Reset() types.PartitionedSystemState {
	s := p.RaftEnvironment.Reset()
	return s.(*RaftState)
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
