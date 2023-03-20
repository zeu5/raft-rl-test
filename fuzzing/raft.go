package fuzzing

import (
	"io"
	"log"

	"go.etcd.io/raft/v3"
	pb "go.etcd.io/raft/v3/raftpb"
)

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
}

func NewRaftEnvironment(config RaftEnvironmentConfig) *RaftEnvironment {
	r := &RaftEnvironment{
		config:   config,
		nodes:    make(map[uint64]*raft.RawNode),
		storages: make(map[uint64]*raft.MemoryStorage),
	}
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
}

func (r *RaftEnvironment) Reset() []pb.Message {
	messages := []pb.Message{{
		Type: pb.MsgProp,
		From: uint64(0),
		Entries: []pb.Entry{
			{Data: []byte("testing")},
		},
	}}
	r.makeNodes()
	return messages
}

func (r *RaftEnvironment) Step(ctx *FuzzContext, m pb.Message) []pb.Message {
	result := make([]pb.Message, 0)
	if m.Type == pb.MsgProp {
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
			m.To = leader
			r.nodes[leader].Step(m)
		} else {
			result = append(result, m)
		}
	} else {
		node := r.nodes[m.To]
		node.Step(m)
	}

	// Take random number of ticks and update node states
	for _, node := range r.nodes {
		ticks := ctx.RandomIntegerChoice(r.config.ElectionTick)
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
			result = append(result, ready.Messages...)
			node.Advance(ready)
		}
	}
	return result
}
