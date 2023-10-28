package redisraft

import (
	"context"
	"time"

	"github.com/zeu5/raft-rl-test/types"
)

type RedisClusterState struct {
	NodeStates map[uint64]*RedisNodeState
	Messages   map[string]Message
}

func (r *RedisClusterState) GetReplicaState(id uint64) types.ReplicaState {
	s := r.NodeStates[id]
	return s
}

func (r *RedisClusterState) PendingMessages() map[string]types.Message {
	out := make(map[string]types.Message)
	for k, m := range r.Messages {
		out[k] = m
	}
	return out
}

var _ types.PartitionedSystemState = &RedisClusterState{}

type RedisRaftEnv struct {
	clusterConfig *ClusterConfig
	network       *InterceptNetwork
	cluster       *Cluster

	curState *RedisClusterState
}

// For a given config, should only be instantiated once since it spins up a sever and binds the addr:port
func NewRedisRaftEnv(ctx context.Context, clusterConfig *ClusterConfig) *RedisRaftEnv {
	e := &RedisRaftEnv{
		clusterConfig: clusterConfig,
		network:       NewInterceptNetwork(ctx, clusterConfig.InterceptListenAddr),
		cluster:       nil,
	}
	e.network.Start()
	return e
}

func (r *RedisRaftEnv) DeliverMessage(m types.Message) types.PartitionedSystemState {
	rm, ok := m.(Message)
	if !ok {
		return r.curState
	}
	r.network.SendMessage(rm.ID)
	newState := &RedisClusterState{
		NodeStates: make(map[uint64]*RedisNodeState),
	}
	for id, s := range r.curState.NodeStates {
		newState.NodeStates[id] = s.Copy()
	}
	newState.Messages = r.network.GetAllMessages()
	r.curState = newState

	return newState
}

func (r *RedisRaftEnv) DropMessage(m types.Message) types.PartitionedSystemState {
	rm, ok := m.(Message)
	if !ok {
		return r.curState
	}
	r.network.DeleteMessage(rm.ID)
	newState := &RedisClusterState{
		NodeStates: make(map[uint64]*RedisNodeState),
	}
	for id, s := range r.curState.NodeStates {
		newState.NodeStates[id] = s.Copy()
	}
	newState.Messages = r.network.GetAllMessages()
	r.curState = newState

	return newState
}

func (r *RedisRaftEnv) Reset() types.PartitionedSystemState {
	if r.cluster != nil {
		r.cluster.Destroy()
	}
	r.network.Reset()
	r.cluster = NewCluster(r.clusterConfig)
	r.cluster.Start()

	newState := &RedisClusterState{
		NodeStates: r.cluster.GetNodeStates(),
		Messages:   r.network.GetAllMessages(),
	}
	r.curState = newState
	return newState
}

func (r *RedisRaftEnv) Tick() types.PartitionedSystemState {
	time.Sleep(5 * time.Millisecond)
	newState := &RedisClusterState{
		NodeStates: r.cluster.GetNodeStates(),
		Messages:   r.network.GetAllMessages(),
	}
	r.curState = newState
	return newState
}

var _ types.PartitionedSystemEnvironment = &RedisRaftEnv{}
