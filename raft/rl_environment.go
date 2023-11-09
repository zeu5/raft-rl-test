package raft

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"sort"
	"strconv"
	"time"

	"github.com/zeu5/raft-rl-test/types"
	raft "go.etcd.io/raft/v3"
	pb "go.etcd.io/raft/v3/raftpb"
	"go.etcd.io/raft/v3/tracker"
)

// Abstraction over different possible raft environment states
// used to analyze coverage
type RaftStateType interface {
	GetNodeStates() map[uint64]raft.Status
}

// State of the Raft environment
type RaftState struct {
	// States of each node (obtained from the raft implementation)
	NodeStates map[uint64]raft.Status
	// The messages in transit
	Messages  map[string]pb.Message
	Logs      map[uint64][]pb.Entry
	Snapshots map[uint64]pb.SnapshotMetadata
	// Test harness and pending requests
	Requests []pb.Message
	// Boolean to indicate if the actions include dropping messages
	WithTimeouts bool
}

// State of a replica in the Raft environment
type RaftReplicaState struct {
	State    raft.Status
	Log      []pb.Entry
	Snapshot pb.SnapshotMetadata
}

// Implements the PartitionedSystemState
func (r RaftState) GetReplicaStateOld(rep uint64) types.ReplicaState {
	return r.NodeStates[rep]
}

// Implements the PartitionedSystemState
func (r RaftState) GetReplicaState(rep uint64) types.ReplicaState {
	return RaftReplicaState{
		State:    r.NodeStates[rep],
		Log:      r.Logs[rep],
		Snapshot: r.Snapshots[rep],
	}
}

// Implements the PartitionedSystemState
func (r RaftState) PendingMessages() map[string]types.Message {
	messages := make(map[string]types.Message)
	for k, m := range r.Messages {
		messages[k] = RaftMessageWrapper{m}
	}
	return messages
}

func (r RaftState) PendingRequests() []types.Request {
	requests := make([]types.Request, len(r.Requests))
	for i, r := range r.Requests {
		requests[i] = copyMessage(r)
	}
	return requests
}

func (r RaftState) CanDeliverRequest() bool {
	haveLeader := false
	for _, s := range r.NodeStates {
		if s.RaftState == raft.StateLeader {
			haveLeader = true
			break
		}
	}
	return haveLeader
}

var _ types.State = RaftState{}
var _ RaftStateType = RaftState{}

func (r RaftState) GetNodeStates() map[uint64]raft.Status {
	return copyNodeStates(r.NodeStates)
}

// Deterministic Hashing, sorting the keys of the replicas before hashing the status
func (r RaftState) MarshalJSON() ([]byte, error) {
	// This function is invoked if json.Marshal(RaftState) is called
	marshalStatus := func(s raft.Status) string {
		j := fmt.Sprintf(`{"id":"%x","term":%d,"vote":"%x","commit":%d,"lead":"%x","raftState":%q,"applied":%d,"progress":{`,
			s.ID, s.Term, s.Vote, s.Commit, s.Lead, s.RaftState, s.Applied)

		if len(s.Progress) == 0 {
			j += "},"
		} else {
			keys := make([]int, 0)
			for k := range s.Progress {
				keys = append(keys, int(k))
			}
			sort.Ints(keys)
			for _, k := range keys {
				v := s.Progress[uint64(k)]
				subj := fmt.Sprintf(`"%d":{"match":%d,"next":%d,"state":%q},`, k, v.Match, v.Next, v.State)
				j += subj
			}
			// remove the trailing ","
			j = j[:len(j)-1] + "},"
		}

		j += fmt.Sprintf(`"leadtransferee":"%x"}`, s.LeadTransferee)
		return j
	}

	res := `{"NodeStates":{`
	keys := make([]int, 0)
	for k := range r.NodeStates {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	for _, k := range keys {
		subS := fmt.Sprintf(`"%x":%s,`, k, marshalStatus(r.NodeStates[uint64(k)]))
		res += subS
	}
	res = res[:len(res)-1] + `},"Messages":`
	if len(r.Messages) == 0 {
		res += "{}}"
	} else {
		bs, _ := json.Marshal(r.Messages)
		res += string(bs) + "}"
	}
	return []byte(res), nil
}

// Deterministic due to custom MarshalJSON
func (r RaftState) Hash() string {
	data, _ := json.Marshal(r)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// to deliver messages or drop all messages based on the receiving replica ids
func (r RaftState) Actions() []types.Action {
	additional := make([]types.Action, 0)
	if r.WithTimeouts {
		processes := map[uint64]bool{}
		for _, m := range r.Messages {
			if m.To != 0 {
				processes[m.To] = true
			}
		}
		for p := range processes {
			additional = append(additional, &RaftAction{
				Type:    "TimeoutProcess",
				Replica: p,
			})
		}
	}
	result := make([]types.Action, len(r.Messages))
	i := 0
	for _, m := range r.Messages {
		result[i] = &RaftAction{
			Type:    "DeliverMessage",
			Message: m,
		}
		i++
	}
	return append(result, additional...)
}

type RaftAction struct {
	Type    string
	Message pb.Message
	Replica uint64
}

func (r *RaftAction) Hash() string {
	data, _ := json.Marshal(r)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

var _ types.Action = &RaftAction{}

// config of the raft environment
type RaftEnvironmentConfig struct {
	Replicas      int
	ElectionTick  int
	HeartbeatTick int
	Timeouts      bool
	Requests      int
}

func (r *RaftEnvironmentConfig) String() string {
	result := "RaftEnvironmentConfig: \n"
	result = fmt.Sprintf("%s Replicas: %d\n", result, r.Replicas)
	result = fmt.Sprintf("%s ElectionTick: %d\n", result, r.ElectionTick)
	result = fmt.Sprintf("%s HeartbeatTick: %d\n", result, r.HeartbeatTick)
	result = fmt.Sprintf("%s Timeouts: %t\n", result, r.Timeouts)
	result = fmt.Sprintf("%s Requests: %d\n\n", result, r.Requests)
	return result
}

// config of the RL dynamics
type RLConfig struct {
	TicksBetweenPartition int  // ticks between two action
	MaxMessagesPerTick    int  // upper bound of messages delivered per tick
	StaySameStateUpTo     int  // amount of consecutive steps in the same partitioning to be considered different
	WithCrashes           bool // enable crash actions
}

func (r *RLConfig) String() string {
	result := "RLConfig: \n"
	result = fmt.Sprintf("%s TicksBetweenPartition: %d\n", result, r.TicksBetweenPartition)
	result = fmt.Sprintf("%s MaxMessagesPerTick: %d\n", result, r.MaxMessagesPerTick)
	result = fmt.Sprintf("%s StaySameStateUpTo: %d\n", result, r.StaySameStateUpTo)
	result = fmt.Sprintf("%s WithCrashes: %t\n\n", result, r.WithCrashes)

	return result
}

// Implements "types.Environment"
type RaftEnvironment struct {
	config   RaftEnvironmentConfig
	nodes    map[uint64]*raft.RawNode
	storages map[uint64]*raft.MemoryStorage
	messages map[string]pb.Message
	curState RaftState
	rand     *rand.Rand
}

func NewRaftEnvironment(config RaftEnvironmentConfig) *RaftEnvironment {
	r := &RaftEnvironment{
		config:   config,
		nodes:    make(map[uint64]*raft.RawNode),
		storages: make(map[uint64]*raft.MemoryStorage),
		messages: make(map[string]pb.Message),
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	for i := 0; i < r.config.Requests; i++ {
		proposal := pb.Message{
			Type: pb.MsgProp,
			From: uint64(0),
			Entries: []pb.Entry{
				{Data: []byte(strconv.Itoa(i + 1))},
			},
		}
		r.messages[msgKey(proposal)] = proposal
	}
	r.makeNodes()
	return r
}

// Creates the replicas and the initial state of the environment
func (r *RaftEnvironment) makeNodes() {
	peers := make([]raft.Peer, r.config.Replicas)
	for i := 0; i < r.config.Replicas; i++ {
		peers[i] = raft.Peer{ID: uint64(i + 1)}
	}
	for i := 0; i < r.config.Replicas; i++ {
		storage := raft.NewMemoryStorage()
		nodeID := uint64(i + 1)
		r.storages[nodeID] = storage
		r.nodes[nodeID], _ = raft.NewRawNode(&raft.Config{
			ID:                        nodeID,
			ElectionTick:              r.config.ElectionTick,
			HeartbeatTick:             r.config.HeartbeatTick,
			Storage:                   storage,
			MaxSizePerMsg:             1024 * 1024,
			MaxInflightMsgs:           256,
			MaxUncommittedEntriesSize: 1 << 30,
			Logger:                    &raft.DefaultLogger{Logger: log.New(io.Discard, "", 0)},
		})
		r.nodes[nodeID].Bootstrap(peers)
	}
	initState := RaftState{
		NodeStates:   make(map[uint64]raft.Status),
		Messages:     copyMessages(r.messages),
		Logs:         make(map[uint64][]pb.Entry),
		Snapshots:    make(map[uint64]pb.SnapshotMetadata),
		WithTimeouts: r.config.Timeouts,
		Requests:     make([]pb.Message, r.config.Requests),
	}
	for i := 0; i < r.config.Requests; i++ {
		initState.Requests[i] = pb.Message{
			Type: pb.MsgProp,
			From: uint64(0),
			Entries: []pb.Entry{
				{Data: []byte(strconv.Itoa(i + 1))},
			},
		}
	}

	for id, node := range r.nodes {
		initState.NodeStates[id] = node.Status()
		initState.Logs[id] = make([]pb.Entry, 0)
	}
	r.curState = initState
}

func (r *RaftEnvironment) Start(node uint64) {
	_, exists := r.nodes[node]
	if exists {
		return
	}
	r.nodes[node], _ = raft.NewRawNode(&raft.Config{
		ID:                        node,
		ElectionTick:              r.config.ElectionTick,
		HeartbeatTick:             r.config.HeartbeatTick,
		Storage:                   r.storages[node],
		MaxSizePerMsg:             1024 * 1024,
		MaxInflightMsgs:           256,
		MaxUncommittedEntriesSize: 1 << 30,
		Logger:                    &raft.DefaultLogger{Logger: log.New(io.Discard, "", 0)},
	})
	peers := make([]raft.Peer, r.config.Replicas)
	for i := 0; i < r.config.Replicas; i++ {
		peers[i] = raft.Peer{ID: uint64(i + 1)}
	}
	r.nodes[node].Bootstrap(peers)
}

func (r *RaftEnvironment) Stop(node uint64) {
	delete(r.nodes, node)
}

func (r *RaftEnvironment) Reset() types.State {
	r.messages = make(map[string]pb.Message)
	r.makeNodes()
	return r.curState
}

// this is probably not called by the partition environment...
func (r *RaftEnvironment) Step(action types.Action) types.State {
	raftAction := action.(*RaftAction)
	switch raftAction.Type {
	case "DeliverMessage":
		// MsgProp - test harness messages are handled specially
		// These messages need to be delivered to the leader
		// If there is no leader then just don't do anything
		if raftAction.Message.Type == pb.MsgProp {
			haveLeader := false
			leader := uint64(0)
			for id, node := range r.nodes {
				if node.Status().RaftState == raft.StateLeader {
					haveLeader = true
					leader = id
					break
				}
			}
			if haveLeader {
				message := raftAction.Message
				message.To = leader
				r.nodes[leader].Step((message))
				delete(r.messages, msgKey(message))
			}
		} else {
			node := r.nodes[raftAction.Message.To]
			node.Step(raftAction.Message)
			delete(r.messages, msgKey(raftAction.Message))
		}
		// Take random number of ticks and update node states
		for _, node := range r.nodes {
			node.Tick()
		}
		// Update the state and return it
		newState := RaftState{
			NodeStates:   make(map[uint64]raft.Status),
			WithTimeouts: r.config.Timeouts,
		}
		for id, node := range r.nodes {
			if node.HasReady() {
				ready := node.Ready()
				if !raft.IsEmptySnap(ready.Snapshot) {
					r.storages[id].ApplySnapshot(ready.Snapshot)
				}
				r.storages[id].Append(ready.Entries)
				// Checking for new messages
				for _, message := range ready.Messages {
					r.messages[msgKey(message)] = message
				}
				node.Advance(ready)
			}
			newState.NodeStates[id] = node.Status()
		}
		newState.Messages = copyMessages(r.messages)
		r.curState = newState
		return newState
	case "TimeoutProcess":
		newMessages := make(map[string]pb.Message)
		for key, message := range r.messages {
			if message.To != raftAction.Replica {
				newMessages[key] = message
			}
		}
		newState := RaftState{
			NodeStates:   copyNodeStates(r.curState.NodeStates),
			Messages:     copyMessages(newMessages),
			WithTimeouts: r.config.Timeouts,
		}
		r.curState = newState
		return newState
	}
	return nil
}

func copyNodeStates(nodeStates map[uint64]raft.Status) map[uint64]raft.Status {
	c := make(map[uint64]raft.Status)
	for k, s := range nodeStates {
		newStatus := raft.Status{
			BasicStatus: raft.BasicStatus{
				ID: s.ID,
				HardState: pb.HardState{
					Term:   s.Term,
					Vote:   s.Vote,
					Commit: s.Commit,
				},
				SoftState: raft.SoftState{
					Lead:      s.Lead,
					RaftState: s.RaftState,
				},
				Applied:        s.Applied,
				LeadTransferee: s.LeadTransferee,
			},
			Config:   s.Config.Clone(),
			Progress: make(map[uint64]tracker.Progress),
		}
		for k, p := range s.Progress {
			newStatus.Progress[k] = tracker.Progress{
				Match:            p.Match,
				Next:             p.Next,
				State:            p.State,
				PendingSnapshot:  p.PendingSnapshot,
				RecentActive:     p.RecentActive,
				MsgAppFlowPaused: p.MsgAppFlowPaused,
				IsLearner:        p.IsLearner,
				Inflights:        p.Inflights.Clone(),
			}
		}
		c[k] = newStatus
	}
	return c
}

func copyMessage(m pb.Message) pb.Message {
	newMessage := pb.Message{
		Type:       m.Type,
		To:         m.To,
		From:       m.From,
		Term:       m.Term,
		LogTerm:    m.LogTerm,
		Index:      m.Index,
		Entries:    make([]pb.Entry, len(m.Entries)),
		Commit:     m.Commit,
		Vote:       m.Vote,
		Snapshot:   m.Snapshot,
		Reject:     m.Reject,
		RejectHint: m.RejectHint,
		Context:    m.Context,
		Responses:  m.Responses,
	}
	for i, entry := range m.Entries {
		newMessage.Entries[i] = pb.Entry{
			Term:  entry.Term,
			Index: entry.Index,
			Type:  entry.Type,
			Data:  entry.Data,
		}
	}
	return newMessage
}

func copyMessagesList(messages []pb.Message) []pb.Message {
	newList := make([]pb.Message, len(messages))
	for i, m := range messages {
		newList[i] = copyMessage(m)
	}
	return newList
}

func copyMessages(messages map[string]pb.Message) map[string]pb.Message {
	c := make(map[string]pb.Message)
	for k, m := range messages {
		newMessage := pb.Message{
			Type:       m.Type,
			To:         m.To,
			From:       m.From,
			Term:       m.Term,
			LogTerm:    m.LogTerm,
			Index:      m.Index,
			Entries:    make([]pb.Entry, len(m.Entries)),
			Commit:     m.Commit,
			Vote:       m.Vote,
			Snapshot:   m.Snapshot,
			Reject:     m.Reject,
			RejectHint: m.RejectHint,
			Context:    m.Context,
			Responses:  m.Responses,
		}
		for i, entry := range m.Entries {
			newMessage.Entries[i] = pb.Entry{
				Term:  entry.Term,
				Index: entry.Index,
				Type:  entry.Type,
				Data:  entry.Data,
			}
		}
		c[k] = newMessage
	}
	return c
}

// copy the logs hashmap (list of pb.Entry structs in the raft code)
func copyLogs(logs map[uint64][]pb.Entry) map[uint64][]pb.Entry {
	c := make(map[uint64][]pb.Entry)
	for k, log := range logs {
		newLog := make([]pb.Entry, len(log))
		for i, entry := range log {
			newLog[i] = pb.Entry{
				Term:  entry.Term,
				Index: entry.Index,
				Type:  entry.Type,
				Data:  entry.Data,
			}
		}
		c[k] = newLog
	}
	return c
}

// copy a log (list of pb.Entry structs in the raft code)
func copyLog(log []pb.Entry) []pb.Entry {
	newLog := make([]pb.Entry, len(log))
	for i, entry := range log {
		newLog[i] = pb.Entry{
			Term:  entry.Term,
			Index: entry.Index,
			Type:  entry.Type,
			Data:  entry.Data,
		}
	}

	return newLog
}

// copy the snapshots hashmap
func copySnapshots(snapshots map[uint64]pb.SnapshotMetadata) map[uint64]pb.SnapshotMetadata {
	c := make(map[uint64]pb.SnapshotMetadata)

	for id, sn := range snapshots {
		newSnap := pb.SnapshotMetadata{
			ConfState: sn.ConfState,
			Index:     sn.Index,
			Term:      sn.Term,
		}

		c[id] = newSnap
	}

	return c
}

func msgKey(message pb.Message) string {
	bs, _ := json.Marshal(message)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}
