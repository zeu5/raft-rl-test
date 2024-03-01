package raft

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"

	"github.com/zeu5/raft-rl-test/types"
	"go.etcd.io/raft/v3"
	pb "go.etcd.io/raft/v3/raftpb"
)

type LinkRaftState struct {
	NodeStates   map[uint64]raft.Status
	MessageLinks map[uint64]map[uint64][]pb.Message
	Timeouts     bool
}

var _ types.State = &LinkRaftState{}
var _ RaftStateType = &LinkRaftState{}

func (l *LinkRaftState) GetNodeStates() map[uint64]raft.Status {
	return l.NodeStates
}

func (l *LinkRaftState) Hash() string {
	bs, _ := json.Marshal(l)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

func (l *LinkRaftState) Actions() []types.Action {
	actions := make([]types.Action, 0)
	for from, m := range l.MessageLinks {
		for to := range m {
			actions = append(actions, &LinkAction{
				Action: "Deliver",
				From:   from,
				To:     to,
			})
			if from != 0 && to != 0 && l.Timeouts {
				actions = append(actions, &LinkAction{
					Action: "Drop",
					From:   from,
					To:     to,
				})
			}

		}
	}
	return actions
}

func (l *LinkRaftState) Terminal() bool {
	return false
}

type LinkAction struct {
	Action string
	From   uint64
	To     uint64
}

func (l *LinkAction) Hash() string {
	d, _ := json.Marshal(l)
	return string(d)
}

type LinkRaftEnvironment struct {
	config   RaftEnvironmentConfig
	nodes    map[uint64]*raft.RawNode
	storages map[uint64]*raft.MemoryStorage
	messages map[uint64]map[uint64][]pb.Message
	curState *LinkRaftState
}

var _ types.Environment = &LinkRaftEnvironment{}

func NewLinkRaftEnvironment(config RaftEnvironmentConfig) *LinkRaftEnvironment {
	r := &LinkRaftEnvironment{
		config:   config,
		nodes:    make(map[uint64]*raft.RawNode),
		storages: make(map[uint64]*raft.MemoryStorage),
		messages: make(map[uint64]map[uint64][]pb.Message),
	}
	proposal := pb.Message{
		Type: pb.MsgProp,
		From: uint64(0),
		To:   uint64(0),
		Entries: []pb.Entry{
			{Data: []byte("testing")},
		},
	}
	r.addMessage(proposal)
	r.makeNodes()
	return r
}

func (r *LinkRaftEnvironment) addMessage(m pb.Message) {
	if _, ok := r.messages[m.From]; !ok {
		r.messages[m.From] = make(map[uint64][]pb.Message)
	}
	if _, ok := r.messages[m.From][m.To]; !ok {
		r.messages[m.From][m.To] = make([]pb.Message, 0)
	}
	r.messages[m.From][m.To] = append(r.messages[m.From][m.To], m)
}

func (r *LinkRaftEnvironment) popMessage(from, to uint64) (pb.Message, bool) {
	fromMessages, ok := r.messages[from]
	if !ok {
		return pb.Message{}, false
	}
	toMessages, ok := fromMessages[to]
	if !ok {
		return pb.Message{}, false
	}
	if len(toMessages) == 0 {
		return pb.Message{}, false
	}
	result := toMessages[0]
	r.messages[from][to] = toMessages[1:]
	return result, true
}

func (r *LinkRaftEnvironment) makeNodes() {
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
	initState := &LinkRaftState{
		NodeStates:   make(map[uint64]raft.Status),
		MessageLinks: r.messages,
		Timeouts:     r.config.Timeouts,
	}
	for id, node := range r.nodes {
		initState.NodeStates[id] = node.Status()
	}
	r.curState = initState
}

func (r *LinkRaftEnvironment) Reset(_ *types.EpisodeContext) (types.State, error) {
	r.messages = make(map[uint64]map[uint64][]pb.Message)
	proposal := pb.Message{
		Type: pb.MsgProp,
		From: uint64(0),
		Entries: []pb.Entry{
			{Data: []byte("testing")},
		},
	}
	r.addMessage(proposal)
	r.makeNodes()
	return r.curState, nil
}

func (r *LinkRaftEnvironment) Step(action types.Action, _ *types.StepContext) (types.State, error) {
	linkAction := action.(*LinkAction)
	message, ok := r.popMessage(linkAction.From, linkAction.To)
	if !ok {
		return r.curState, nil
	}
	switch linkAction.Action {
	case "Deliver":
		if message.Type == pb.MsgProp {
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
			} else {
				r.addMessage(message)
			}
		} else {
			node := r.nodes[message.To]
			node.Step(message)
		}
		// Take random number of ticks and update node states
		for _, node := range r.nodes {
			ticks := 6
			for i := 0; i < ticks; i++ {
				node.Tick()
			}
		}
		newState := &LinkRaftState{
			NodeStates: make(map[uint64]raft.Status),
			Timeouts:   r.config.Timeouts,
		}
		for id, node := range r.nodes {
			if node.HasReady() {
				ready := node.Ready()
				if !raft.IsEmptySnap(ready.Snapshot) {
					r.storages[id].ApplySnapshot(ready.Snapshot)
				}
				r.storages[id].Append(ready.Entries)
				for _, message := range ready.Messages {
					r.addMessage(message)
				}
				node.Advance(ready)
			}
			newState.NodeStates[id] = node.Status()
		}
		newState.MessageLinks = r.messages
		r.curState = newState
		return newState, nil
	case "Drop":
		newState := &LinkRaftState{
			NodeStates:   r.curState.NodeStates,
			MessageLinks: r.messages,
		}
		r.curState = newState
		return newState, nil
	}
	return nil, nil
}

func (r *LinkRaftEnvironment) StepCtx(action types.Action, timeoutCtx context.Context) types.State {
	linkAction := action.(*LinkAction)
	message, ok := r.popMessage(linkAction.From, linkAction.To)
	if !ok {
		return r.curState
	}
	switch linkAction.Action {
	case "Deliver":
		if message.Type == pb.MsgProp {
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
			} else {
				r.addMessage(message)
			}
		} else {
			node := r.nodes[message.To]
			node.Step(message)
		}
		// Take random number of ticks and update node states
		for _, node := range r.nodes {
			ticks := 6
			for i := 0; i < ticks; i++ {
				node.Tick()
			}
		}
		newState := &LinkRaftState{
			NodeStates: make(map[uint64]raft.Status),
			Timeouts:   r.config.Timeouts,
		}
		for id, node := range r.nodes {
			if node.HasReady() {
				ready := node.Ready()
				if !raft.IsEmptySnap(ready.Snapshot) {
					r.storages[id].ApplySnapshot(ready.Snapshot)
				}
				r.storages[id].Append(ready.Entries)
				for _, message := range ready.Messages {
					r.addMessage(message)
				}
				node.Advance(ready)
			}
			newState.NodeStates[id] = node.Status()
		}
		newState.MessageLinks = r.messages
		r.curState = newState
		return newState
	case "Drop":
		newState := &LinkRaftState{
			NodeStates:   r.curState.NodeStates,
			MessageLinks: r.messages,
		}
		r.curState = newState
		return newState
	}
	return nil
}
