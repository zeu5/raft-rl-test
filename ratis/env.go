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

func (r *RatisClusterState) CanDeliverRequest() bool {
	return false
}

func (r *RatisClusterState) PendingRequests() []types.Request {
	return []types.Request{}
}

func (r *RatisClusterState) Copy() *RatisClusterState {
	out := &RatisClusterState{
		NodeStates: make(map[uint64]*RatisNodeState),
		Messages:   make(map[string]Message),
	}
	for k, v := range r.NodeStates {
		out.NodeStates[k] = v.Copy()
	}
	for k, v := range r.Messages {
		out.Messages[k] = v.Copy()
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
		network:       NewInterceptNetwork(ctx, clusterConfig.InterceptListenPort),
		cluster:       nil,
	}
	e.network.Start()
	return e
}

func (r RatisRaftEnv) ReceiveRequest(types.Request) types.PartitionedSystemState {
	newState := r.curState.Copy()
	r.curState = newState
	return newState
}

func (r *RatisRaftEnv) Start(node uint64) {
	// TODO: Need to implement this
	panic("should not come here")
}

func (r *RatisRaftEnv) Stop(node uint64) {
	// TODO: Need to implement this
	panic("should not come here")
}

func (r *RatisRaftEnv) DeliverMessages(messages []types.Message) types.PartitionedSystemState {
	newState := r.curState.Copy()

	for _, m := range messages {
		rm, ok := m.(Message)
		if !ok {
			continue
		}
		r.network.SendMessage(rm.ID)
	}
	newState.Messages = r.network.GetAllMessages()
	r.curState = newState
	return newState
}

func (r *RatisRaftEnv) DropMessages(messages []types.Message) types.PartitionedSystemState {

	newState := &RatisClusterState{
		NodeStates: make(map[uint64]*RatisNodeState),
	}
	for id, s := range r.curState.NodeStates {
		newState.NodeStates[id] = s.Copy()
	}
	for _, m := range messages {
		rm, ok := m.(Message)
		if !ok {
			continue
		}
		r.network.DeleteMessage(rm.ID)
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

	r.network.WaitForNodes(r.clusterConfig.NumNodes)

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
	time.Sleep(20 * time.Millisecond)
	newState := &RatisClusterState{
		NodeStates: r.cluster.GetNodeStates(),
		Messages:   r.network.GetAllMessages(),
	}
	r.curState = newState
	return newState
}

var _ types.PartitionedSystemEnvironment = &RatisRaftEnv{}
