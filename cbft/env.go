package cbft

import (
	"context"
	"time"

	"github.com/zeu5/raft-rl-test/types"
)

type CometClusterState struct {
	NodeStates map[uint64]*CometNodeState
	Messages   map[string]Message
}

func (r *CometClusterState) GetReplicaState(id uint64) types.ReplicaState {
	s := r.NodeStates[id]
	return s
}

func (r *CometClusterState) PendingMessages() map[string]types.Message {
	out := make(map[string]types.Message)
	for k, m := range r.Messages {
		out[k] = m
	}
	return out
}

func (r *CometClusterState) CanDeliverRequest() bool {
	return false
}

func (r *CometClusterState) PendingRequests() []types.Request {
	return []types.Request{}
}

func (r *CometClusterState) Copy() *CometClusterState {
	out := &CometClusterState{
		NodeStates: make(map[uint64]*CometNodeState),
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

var _ types.PartitionedSystemState = &CometClusterState{}

type CometEnv struct {
	clusterConfig *CometClusterConfig
	network       *InterceptNetwork
	cluster       *CometCluster

	curState *CometClusterState
}

// For a given config, should only be instantiated once since it spins up a sever and binds the addr:port
func NewCometEnv(ctx context.Context, clusterConfig *CometClusterConfig) *CometEnv {
	e := &CometEnv{
		clusterConfig: clusterConfig,
		network:       NewInterceptNetwork(ctx, clusterConfig.InterceptListenPort),
		cluster:       nil,
	}
	e.network.Start()
	return e
}

func (r CometEnv) ReceiveRequest(types.Request) types.PartitionedSystemState {
	newState := r.curState.Copy()

	r.curState = newState
	return newState
}

func (r *CometEnv) Start(node uint64) {
	// TODO: Need to implement this
	panic("should not come here")
}

func (r *CometEnv) Stop(node uint64) {
	// TODO: Need to implement this
	panic("should not come here")
}

func (r *CometEnv) DeliverMessages(messages []types.Message) types.PartitionedSystemState {
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

func (r *CometEnv) DropMessages(messages []types.Message) types.PartitionedSystemState {

	newState := &CometClusterState{
		NodeStates: make(map[uint64]*CometNodeState),
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

func (r *CometEnv) Reset() types.PartitionedSystemState {
	if r.cluster != nil {
		r.cluster.Destroy()
	}
	r.network.Reset()
	r.cluster, _ = NewCluster(r.clusterConfig)
	r.cluster.Start()

	r.network.WaitForNodes(r.clusterConfig.NumNodes)

	newState := &CometClusterState{
		NodeStates: r.cluster.GetNodeStates(),
		Messages:   r.network.GetAllMessages(),
	}
	r.curState = newState
	return newState
}

func (r *CometEnv) Cleanup() {
	if r.cluster != nil {
		r.cluster.Destroy()
		r.cluster = nil
	}
}

func (r *CometEnv) Tick() types.PartitionedSystemState {
	time.Sleep(20 * time.Millisecond)
	newState := &CometClusterState{
		NodeStates: r.cluster.GetNodeStates(),
		Messages:   r.network.GetAllMessages(),
	}
	r.curState = newState
	return newState
}

var _ types.PartitionedSystemEnvironment = &CometEnv{}
