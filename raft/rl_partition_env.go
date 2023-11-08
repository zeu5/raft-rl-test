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
type RaftColorFunc func(RaftReplicaState) (string, interface{})

// return the state of the node, should be one of these:
//
//	var stmap = [...]string{
//		"StateFollower",
//		"StateCandidate",
//		"StateLeader",
//		"StatePreCandidate",
//	}
func ColorState() RaftColorFunc {
	return func(s RaftReplicaState) (string, interface{}) {
		return "state", s.State.RaftState.String()
	}
}

// return the current term of the node, should be just a number
func ColorTerm() RaftColorFunc {
	return func(s RaftReplicaState) (string, interface{}) {
		return "term", s.State.Term
	}
}

// return the current term of the node, provides a bound that is the maximum considered term
func ColorBoundedTerm(bound int) RaftColorFunc {
	return func(s RaftReplicaState) (string, interface{}) {
		term := s.State.Term
		if term > uint64(bound) {
			term = uint64(bound)
		}
		return "boundedTerm", term
	}
}

func ColorCommit() RaftColorFunc {
	return func(s RaftReplicaState) (string, interface{}) {
		return "commit", s.State.Commit
	}
}

func ColorApplied() RaftColorFunc {
	return func(s RaftReplicaState) (string, interface{}) {
		return "applied", s.State.Applied
	}
}

func ColorVote() RaftColorFunc {
	return func(s RaftReplicaState) (string, interface{}) {
		return "vote", s.State.Vote
	}
}

func ColorLeader() RaftColorFunc {
	return func(s RaftReplicaState) (string, interface{}) {
		return "leader", s.State.Lead
	}
}

// return the snapshot index of a replica
func ColorSnapshotIndex() RaftColorFunc {
	return func(s RaftReplicaState) (string, interface{}) {
		return "snapshotIndex", s.Snapshot.Index
	}
}

// return the snapshot index of a replica
func ColorSnapshotTerm() RaftColorFunc {
	return func(s RaftReplicaState) (string, interface{}) {
		return "snapshotTerm", s.Snapshot.Term
	}
}

// return the snapshot index of a replica
func ColorReplicaID() RaftColorFunc {
	return func(s RaftReplicaState) (string, interface{}) {
		return "replicaID", s.State.ID
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
	replicaState := s.(RaftReplicaState) // cast into RaftReplicaState
	c := &RaftPartitionColor{
		Params: make(map[string]interface{}),
	}

	for _, p := range p.paramFuncs {
		k, v := p(replicaState)
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
		Snapshots:    make(map[uint64]pb.SnapshotMetadata),
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

		// add status
		status := node.Status()
		newState.NodeStates[id] = status

		// add log
		newState.Logs[id] = make([]pb.Entry, 0)
		ents, err := p.storages[id].Entries(1, status.Commit+1, 1024*1024) // hardcoded value from link_env.go
		if err == nil {
			newState.Logs[id] = ents
		}

		// add snapshot
		snapshot, err := p.storages[id].Snapshot()
		if err == nil {
			newState.Snapshots[id] = snapshot.Metadata
		}
	}
	newState.Messages = copyMessages(p.messages)
	newState.Requests = copyMessagesList(p.curState.Requests)
	p.curState = newState
	return newState
}

func (p *RaftPartitionEnv) ReceiveRequest(r types.Request) types.PartitionedSystemState {
	newState := RaftState{
		NodeStates:   copyNodeStates(p.curState.NodeStates),
		Messages:     copyMessages(p.curState.Messages),
		Logs:         copyLogs(p.curState.Logs), // copy the logs for the new state
		Snapshots:    copySnapshots(p.curState.Snapshots),
		WithTimeouts: p.config.Timeouts,
		Requests:     make([]pb.Message, 0),
	}

	haveLeader := false
	leader := uint64(0)
	for id, node := range p.nodes {
		if node.Status().RaftState == raft.StateLeader {
			haveLeader = true
			leader = id
			break
		}
	}
	remainingRequests := p.curState.Requests
	if haveLeader {
		message := r.(pb.Message)
		message.To = leader
		p.nodes[leader].Step((message))
		remainingRequests = p.curState.Requests[1:]
	}
	for _, r := range remainingRequests {
		newState.Requests = append(newState.Requests, copyMessage(r))
	}
	p.curState = newState
	return newState
}

func (l *RaftPartitionEnv) DeliverMessages(messages []types.Message) types.PartitionedSystemState {
	var s types.PartitionedSystemState = nil
	for _, m := range messages {
		s = l.deliverMessage(m)
	}
	return s
}

// deliver the specified message in the system and returns the subsequent state, no tick pass?
func (p *RaftPartitionEnv) deliverMessage(m types.Message) types.PartitionedSystemState {
	rm := m.(RaftMessageWrapper)
	node, exists := p.nodes[rm.Message.To]
	if exists {
		node.Step(rm.Message)
	}
	delete(p.messages, msgKey(rm.Message))

	newState := RaftState{
		NodeStates:   make(map[uint64]raft.Status),
		WithTimeouts: p.config.Timeouts,
		Requests:     copyMessagesList(p.curState.Requests),
		Logs:         make(map[uint64][]pb.Entry),
		Snapshots:    make(map[uint64]pb.SnapshotMetadata),
	}
	for id, node := range p.nodes {
		if node.HasReady() {
			ready := node.Ready()
			if !raft.IsEmptySnap(ready.Snapshot) {
				p.storages[id].ApplySnapshot(ready.Snapshot)
				// snap, err := p.storages[id].Snapshot()
			}
			if len(ready.Entries) > 0 {
				p.storages[id].Append(ready.Entries)
			}
			for _, message := range ready.Messages {
				p.messages[msgKey(message)] = message
			}
			node.Advance(ready)
		}
		// add status
		status := node.Status()
		newState.NodeStates[id] = status

		// add log
		newState.Logs[id] = make([]pb.Entry, 0)
		ents, err := p.storages[id].Entries(1, status.Commit+1, 1024*1024) // hardcoded value from link_env.go
		if err == nil {
			newState.Logs[id] = ents
		}

		// add snapshot
		snapshot, err := p.storages[id].Snapshot()
		if err == nil {
			newState.Snapshots[id] = snapshot.Metadata
		}

	}
	newState.Messages = copyMessages(p.messages)
	p.curState = newState
	return newState
}

func (l *RaftPartitionEnv) DropMessages(messages []types.Message) types.PartitionedSystemState {
	var s types.PartitionedSystemState
	for _, m := range messages {
		s = l.dropMessage(m)
	}
	return s
}

// drops the specified message in the system, no tick pass
func (p *RaftPartitionEnv) dropMessage(m types.Message) types.PartitionedSystemState {
	newState := RaftState{
		NodeStates:   copyNodeStates(p.curState.NodeStates),
		Messages:     copyMessages(p.curState.Messages),
		Logs:         copyLogs(p.curState.Logs), // copy the logs for the new state
		Snapshots:    copySnapshots(p.curState.Snapshots),
		Requests:     copyMessagesList(p.curState.Requests),
		WithTimeouts: p.config.Timeouts,
	}
	delete(newState.Messages, m.Hash())
	p.curState = newState
	return newState
}
