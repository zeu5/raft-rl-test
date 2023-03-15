package raft

import (
	"encoding/json"
	"io"
	"log"
	"math/rand"
	"time"

	"github.com/zeu5/raft-rl-test/types"
	raft "go.etcd.io/raft/v3"
	pb "go.etcd.io/raft/v3/raftpb"
)

type RaftState struct {
	NodeStates   map[uint64]raft.Status
	Messages     map[string]pb.Message
	WithTimeouts bool
}

var _ types.State = &RaftState{}

func (r *RaftState) Hash() string {
	data, _ := json.Marshal(r)
	return string(data)
}

func (r *RaftState) Actions() []types.Action {
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
	return ""
}

var _ types.Action = &RaftAction{}

type RaftEnvironmentConfig struct {
	Replicas      int
	ElectionTick  int
	HeartbeatTick int
	Timeouts      bool
}

type RaftEnvironment struct {
	config   RaftEnvironmentConfig
	nodes    map[uint64]*raft.RawNode
	storages map[uint64]*raft.MemoryStorage
	messages map[string]pb.Message
	curState *RaftState
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
	proposal := pb.Message{
		Type: pb.MsgProp,
		From: uint64(0),
		Entries: []pb.Entry{
			{Data: []byte("testing")},
		},
	}
	r.messages[proposal.String()] = proposal
	r.makeNodes()
	return r
}

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
	initState := &RaftState{
		NodeStates:   make(map[uint64]raft.Status),
		Messages:     r.messages,
		WithTimeouts: r.config.Timeouts,
	}
	for id, node := range r.nodes {
		initState.NodeStates[id] = node.Status()
	}
	r.curState = initState
}

func (r *RaftEnvironment) Reset() types.State {
	r.messages = make(map[string]pb.Message)
	proposal := pb.Message{
		Type: pb.MsgProp,
		From: uint64(0),
		Entries: []pb.Entry{
			{Data: []byte("testing")},
		},
	}
	r.messages[proposal.String()] = proposal
	r.makeNodes()
	return r.curState
}

func (r *RaftEnvironment) Step(action types.Action) types.State {
	raftAction := action.(*RaftAction)
	switch raftAction.Type {
	case "DeliverMessage":
		if raftAction.Message.Type == pb.MsgProp {
			// TODO: handle proposal separately
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
				delete(r.messages, message.String())
			}
		} else {
			node := r.nodes[raftAction.Message.To]
			node.Step(raftAction.Message)
			delete(r.messages, raftAction.Message.String())
		}
		// Take random number of ticks and update node states
		for _, node := range r.nodes {
			ticks := 6
			for i := 0; i < ticks; i++ {
				node.Tick()
			}
		}
		newState := &RaftState{
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
				for _, message := range ready.Messages {
					r.messages[message.String()] = message
				}
				node.Advance(ready)
			}
			newState.NodeStates[id] = node.Status()
		}
		newState.Messages = r.messages
		r.curState = newState
		return newState
	case "TimeoutProcess":
		newMessages := make(map[string]pb.Message)
		for key, message := range r.messages {
			if message.To != raftAction.Replica {
				newMessages[key] = message
			}
		}
		newState := &RaftState{
			NodeStates:   r.curState.NodeStates,
			Messages:     newMessages,
			WithTimeouts: r.config.Timeouts,
		}
		r.curState = newState
		return newState
	}
	return nil
}
