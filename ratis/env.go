package ratis

import (
	"context"
	"time"

	"github.com/zeu5/raft-rl-test/types"
)

type RatisClusterState struct {
	NodeStates map[uint64]*RatisNodeState
	Messages   map[string]Message
}

func (r *RatisClusterState) GetReplicaState(id uint64) types.ReplicaState {
	s := r.NodeStates[id]
	return s
}

func (r *RatisClusterState) PendingMessages() map[string]types.Message {
	out := make(map[string]types.Message)
	for k, m := range r.Messages {
		out[k] = m
	}
	return out
}

var _ types.PartitionedSystemState = &RatisClusterState{}

type RatisRaftEnv struct {
	clusterConfig *RatisClusterConfig
	network       *InterceptNetwork
	cluster       *RatisCluster

	curState *RatisClusterState
}

// For a given config, should only be instantiated once since it spins up a sever and binds the addr:port
func NewRatisRaftEnv(ctx context.Context, clusterConfig *RatisClusterConfig) *RatisRaftEnv {
	e := &RatisRaftEnv{
		clusterConfig: clusterConfig,
		network:       NewInterceptNetwork(ctx, clusterConfig.InterceptListenAddr),
		cluster:       nil,
	}
	e.network.Start()
	return e
}

func (r *RatisRaftEnv) DeliverMessage(m types.Message) types.PartitionedSystemState {
	rm, ok := m.(Message)
	if !ok {
		return r.curState
	}
	r.network.SendMessage(rm.ID)
	newState := &RatisClusterState{
		NodeStates: make(map[uint64]*RatisNodeState),
	}
	for id, s := range r.curState.NodeStates {
		newState.NodeStates[id] = s.Copy()
	}
	newState.Messages = r.network.GetAllMessages()
	r.curState = newState

	return newState
}

func (r *RatisRaftEnv) DropMessage(m types.Message) types.PartitionedSystemState {
	rm, ok := m.(Message)
	if !ok {
		return r.curState
	}
	r.network.DeleteMessage(rm.ID)
	newState := &RatisClusterState{
		NodeStates: make(map[uint64]*RatisNodeState),
	}
	for id, s := range r.curState.NodeStates {
		newState.NodeStates[id] = s.Copy()
	}
	newState.Messages = r.network.GetAllMessages()
	r.curState = newState

	return newState
}

func (r *RatisRaftEnv) Reset() types.PartitionedSystemState {
	if r.cluster != nil {
		r.cluster.Destroy()
	}
	r.network.Reset()
	r.cluster = NewCluster(r.clusterConfig)
	r.cluster.Start()

	newState := &RatisClusterState{
		NodeStates: r.cluster.GetNodeStates(),
		Messages:   r.network.GetAllMessages(),
	}
	r.curState = newState
	return newState
}

func (r *RatisRaftEnv) Cleanup() {
	if r.cluster != nil {
		r.cluster.Destroy()
		r.cluster = nil
	}
}

func (r *RatisRaftEnv) Tick() types.PartitionedSystemState {
	time.Sleep(50 * time.Microsecond)
	newState := &RatisClusterState{
		NodeStates: r.cluster.GetNodeStates(),
		Messages:   r.network.GetAllMessages(),
	}
	r.curState = newState
	return newState
}

var _ types.PartitionedSystemEnvironment = &RatisRaftEnv{}
