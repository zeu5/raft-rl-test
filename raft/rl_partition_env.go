package raft

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/zeu5/raft-rl-test/types"
	"go.etcd.io/raft/v3"
	pb "go.etcd.io/raft/v3/raftpb"
)

// var _ types.ReplicaState = raft.Status{}

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

// various functions to take info from raft.Status to 'colors' of the abstracted state
type RaftColorFunc func(raft.Status) (string, interface{})

// return the state of the node, should be one of these:
//
//	var stmap = [...]string{
//		"StateFollower",
//		"StateCandidate",
//		"StateLeader",
//		"StatePreCandidate",
//	}
func ColorState() RaftColorFunc {
	return func(s raft.Status) (string, interface{}) {
		return "state", s.RaftState.String()
	}
}

// return the current term of the node, should be just a number
func ColorTerm() RaftColorFunc {
	return func(s raft.Status) (string, interface{}) {
		return "term", s.Term
	}
}

// return the current term of the node, provides a bound that is the maximum considered term
func ColorBoundedTerm(bound int) RaftColorFunc {
	return func(s raft.Status) (string, interface{}) {
		term := s.Term
		if term > uint64(bound) {
			term = uint64(bound)
		}
		return "boundedTerm", term
	}
}

// return the relative current term of the node, 1 is the current minimum term number, the others are computed as difference from this
func ColorRelativeTerm(minimum int) RaftColorFunc {
	return func(s raft.Status) (string, interface{}) {
		term := s.Term                    // get the value from process Status
		term = term - uint64(minimum) + 1 // compute difference
		return "boundedTerm", term
	}
}

// return the relative current term of the node, 1 is the current minimum term number, the others are computed as difference from this
// bound provides the maximum considered term gap
func ColorRelativeBoundedTerm(minimum int, bound int) RaftColorFunc {
	return func(s raft.Status) (string, interface{}) {
		term := s.Term                    // get the value from process Status
		term = term - uint64(minimum) + 1 // compute difference
		if term > uint64(bound) {         // set to bound if higher gap
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

// apply abstraction on a ReplicaState
func (p *RaftStatePainter) Color(s types.ReplicaState) types.Color {
	rsState := s.(map[string]interface{}) // cast into a map
	rs := rsState["state"].(raft.Status)  // cast the "state" component into raft.Status
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

// partition environment in types.partition_env.go will call these two functions multiple times inbetween partition actions
// it seems they are not using the step function of the underlying environment, they interact directly with the raft code...

// make one tick pass in the system and returns the subsequent env state
func (p *RaftPartitionEnv) Tick() types.PartitionedSystemState {
	for _, node := range p.nodes {
		node.Tick()
	}
	newState := RaftState{
		NodeStates:   make(map[uint64]raft.Status),
		WithTimeouts: p.config.Timeouts,
		Logs:         make(map[uint64][]pb.Entry), // guess this should be added also here?
	}
	for id, node := range p.nodes {
		if node.HasReady() {
			ready := node.Ready()
			if !raft.IsEmptySnap(ready.Snapshot) {
				p.storages[id].ApplySnapshot(ready.Snapshot)
			}
			if len(ready.Entries) > 0 {
				p.storages[id].Append(ready.Entries)
			}
			for _, message := range ready.Messages {
				p.messages[msgKey(message)] = message
			}
			node.Advance(ready)
		}
		status := node.Status()

		// populate replica log
		newState.Logs[id] = make([]pb.Entry, 0)
		ents, err := p.storages[id].Entries(1, status.Commit+1, 1024*1024) // hardcoded value from link_env.go
		if err == nil {
			newState.Logs[id] = ents
		}

		newState.NodeStates[id] = status
	}
	newState.Messages = copyMessages(p.messages)
	p.curState = newState
	return newState
}

// deliver the specified message in the system and returns the subsequent state, no tick pass?
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
			messageKey := msgKey(message)
			message.To = leader
			p.nodes[leader].Step((message))
			delete(p.messages, messageKey)
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
			if len(ready.Entries) > 0 {
				p.storages[id].Append(ready.Entries)
			}
			for _, message := range ready.Messages {
				p.messages[msgKey(message)] = message
			}
			node.Advance(ready)
		}
		status := node.Status()
		newState.Logs[id] = make([]pb.Entry, 0)
		ents, err := p.storages[id].Entries(1, status.Commit+1, 1024*1024) // hardcoded value from link_env.go
		if err == nil {
			newState.Logs[id] = ents
		}
		newState.NodeStates[id] = status
	}
	newState.Messages = copyMessages(p.messages)
	p.curState = newState
	return newState
}

// drops the specified message in the system, no tick pass
func (p *RaftPartitionEnv) DropMessage(m types.Message) types.PartitionedSystemState {
	newState := RaftState{
		NodeStates:   copyNodeStates(p.curState.NodeStates),
		Messages:     copyMessages(p.curState.Messages),
		Logs:         copyLogs(p.curState.Logs), // copy the logs for the new state
		WithTimeouts: p.config.Timeouts,
	}
	delete(newState.Messages, m.Hash())
	p.curState = newState
	return newState
}
