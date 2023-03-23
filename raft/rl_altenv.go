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

type AbsRaftState struct {
	NodeStates   map[uint64]raft.Status
	Messages     map[uint64]pb.Message
	WithTimeouts bool
}

var _ types.State = &AbsRaftState{}

func (r *AbsRaftState) Hash() string {
	data, _ := json.Marshal(r)
	return string(data)
}

func (r *AbsRaftState) Actions() []types.Action {
	additional := make([]types.Action, 0)
	if r.WithTimeouts {
		processes := map[uint64]bool{}
		for q := range r.Messages {
			if q != 0 {
				processes[q] = true
			}
		}
		for p := range processes {
			additional = append(additional, &AbsRaftAction{
				Type:    "TimeoutProcess",
				Replica: p,
			})
		}
	}
	result := make([]types.Action, len(r.Messages))
	i := 0
	for q := range r.Messages {
		result[i] = &AbsRaftAction{
			Type:    "DeliverMessage",
			Replica: q,
		}
		i++
	}
	return append(result, additional...)
}

type AbsRaftAction struct {
	Type    string
	Replica uint64
}

func (r *AbsRaftAction) Hash() string {
	b, _ := json.Marshal(r)
	return string(b)
}

var _ types.Action = &AbsRaftAction{}

type AbsRaftEnvironment struct {
	config        RaftEnvironmentConfig
	nodes         map[uint64]*raft.RawNode
	storages      map[uint64]*raft.MemoryStorage
	messageQueues map[uint64][]pb.Message
	curState      *AbsRaftState
	rand          *rand.Rand
}

func NewAbsRaftEnvironment(config RaftEnvironmentConfig) *AbsRaftEnvironment {
	r := &AbsRaftEnvironment{
		config:        config,
		nodes:         make(map[uint64]*raft.RawNode),
		storages:      make(map[uint64]*raft.MemoryStorage),
		messageQueues: make(map[uint64][]pb.Message),
		rand:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	proposal := pb.Message{
		Type: pb.MsgProp,
		From: uint64(0),
		Entries: []pb.Entry{
			{Data: []byte("testing")},
		},
	}
	r.messageQueues[0] = make([]pb.Message, 1)
	r.messageQueues[0][0] = proposal
	r.makeNodes()
	return r
}

func (r *AbsRaftEnvironment) makeNodes() {
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
	r.updateState()
}

func (r *AbsRaftEnvironment) Reset() types.State {
	r.messageQueues = make(map[uint64][]pb.Message)
	proposal := pb.Message{
		Type: pb.MsgProp,
		From: uint64(0),
		Entries: []pb.Entry{
			{Data: []byte("testing")},
		},
	}
	r.messageQueues[0] = make([]pb.Message, 1)
	r.messageQueues[0][0] = proposal
	r.makeNodes()
	return r.curState
}

func (r *AbsRaftEnvironment) popQueue(node uint64) {
	queue, ok := r.messageQueues[node]
	if ok && len(queue) > 1 {
		r.messageQueues[node] = queue[1:]
	}
}

func (r *AbsRaftEnvironment) updateState() {
	newState := &AbsRaftState{
		NodeStates:   make(map[uint64]raft.Status),
		Messages:     make(map[uint64]pb.Message),
		WithTimeouts: r.config.Timeouts,
	}
	for id, node := range r.nodes {
		newState.NodeStates[id] = node.Status()
		queue, ok := r.messageQueues[id]
		if ok && len(queue) > 1 {
			newState.Messages[id] = queue[0]
		}
	}
	r.curState = newState
}

func (r *AbsRaftEnvironment) Step(action types.Action) types.State {
	raftAction := action.(*AbsRaftAction)
	switch raftAction.Type {
	case "DeliverMessage":
		queue, ok := r.messageQueues[raftAction.Replica]
		if !ok || len(queue) == 0 {
			return r.curState
		}
		message := queue[0]
		if message.Type == pb.MsgProp {
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
				message.To = leader
				r.nodes[leader].Step((message))
				r.popQueue(raftAction.Replica)
			}
		} else {
			node := r.nodes[message.To]
			node.Step(message)
			r.popQueue(raftAction.Replica)
		}
		// Take random number of ticks and update node states
		for _, node := range r.nodes {
			ticks := 6
			for i := 0; i < ticks; i++ {
				node.Tick()
			}
		}
		for id, node := range r.nodes {
			if node.HasReady() {
				ready := node.Ready()
				if !raft.IsEmptySnap(ready.Snapshot) {
					r.storages[id].ApplySnapshot(ready.Snapshot)
				}
				r.storages[id].Append(ready.Entries)
				for _, message := range ready.Messages {
					_, ok := r.messageQueues[message.To]
					if !ok {
						r.messageQueues[message.To] = make([]pb.Message, 0)
					}
					r.messageQueues[message.To] = append(r.messageQueues[message.To], message)
				}
				node.Advance(ready)
			}
		}
		r.updateState()
		return r.curState
	case "TimeoutProcess":
		r.popQueue(raftAction.Replica)
		r.updateState()
		return r.curState
	}
	return nil
}
