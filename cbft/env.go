package cbft

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/zeu5/raft-rl-test/types"
)

type CometRequest struct {
	// One of "get|set"
	Type string
	// Key of the operation
	Key string
	// Value of the operation in the case of set
	Value string
}

func (c CometRequest) Copy() CometRequest {
	return CometRequest{
		Type:  c.Type,
		Key:   c.Key,
		Value: c.Value,
	}
}

type CometClusterState struct {
	NodeStates map[uint64]*CometNodeState
	Messages   map[string]Message
	Requests   []CometRequest
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
	return len(r.Requests) > 0
}

func (r *CometClusterState) PendingRequests() []types.Request {
	out := make([]types.Request, len(r.Requests))
	for i, r := range r.Requests {
		out[i] = r.Copy()
	}
	return out
}

func (r *CometClusterState) Copy() *CometClusterState {
	out := &CometClusterState{
		NodeStates: make(map[uint64]*CometNodeState),
		Messages:   make(map[string]Message),
		Requests:   make([]CometRequest, 0),
	}
	for k, v := range r.NodeStates {
		out.NodeStates[k] = v.Copy()
	}
	for k, v := range r.Messages {
		out.Messages[k] = v.Copy()
	}
	for i, r := range r.Requests {
		out.Requests[i] = r.Copy()
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

func (r *CometEnv) BecomeByzantine(nodeID uint64) {
	r.network.MakeNodeByzantine(nodeID)
}

func (r *CometEnv) ReceiveRequest(req types.Request) types.PartitionedSystemState {
	newState := r.curState.Copy()

	cReq := req.(CometRequest)
	queryString := ""
	if cReq.Type == "get" {
		queryString = fmt.Sprintf(`abci_query?data="%s"`, cReq.Key)
	} else if cReq.Type == "set" {
		queryString = fmt.Sprintf(`broadcast_tx_commit?tx="%s=%s"`, cReq.Key, cReq.Value)
	}
	if queryString == "" {
		r.curState = newState
		return newState
	}

	r.cluster.Execute(queryString)
	newState.Requests = copyRequests(newState.Requests[1:])

	r.curState = newState
	return newState
}

func (r *CometEnv) Start(nodeID uint64) {
	if r.cluster == nil {
		return
	}
	node, ok := r.cluster.GetNode(int(nodeID))
	if !ok {
		return
	}
	if node.IsActive() {
		return
	}
	node.Start()
}

func (r *CometEnv) Stop(nodeID uint64) {
	if r.cluster == nil {
		return
	}
	node, ok := r.cluster.GetNode(int(nodeID))
	if !ok {
		return
	}
	if !node.IsActive() {
		return
	}
	node.Stop()
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
	newState.Requests = copyRequests(r.curState.Requests)
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
		Requests:   make([]CometRequest, r.clusterConfig.NumRequests),
	}
	for i := 0; i < r.clusterConfig.NumRequests; i++ {
		if rand.Intn(2) == 0 {
			newState.Requests[i] = CometRequest{
				Type: "get",
				Key:  "k",
			}
		} else {
			newState.Requests[i] = CometRequest{
				Type:  "set",
				Key:   "k",
				Value: "v",
			}
		}
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
		Requests:   copyRequests(r.curState.Requests),
	}
	r.curState = newState
	return newState
}

var _ types.PartitionedSystemEnvironment = &CometEnv{}
