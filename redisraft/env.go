package redisraft

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/zeu5/raft-rl-test/types"
)

type RedisRequest struct {
	Type string
}

func (r RedisRequest) Copy() RedisRequest {
	return RedisRequest{
		Type: r.Type,
	}
}

func copyRequests(requests []RedisRequest) []RedisRequest {
	out := make([]RedisRequest, len(requests))
	for i, r := range requests {
		out[i] = r.Copy()
	}
	return out
}

type RedisClusterState struct {
	NodeStates map[uint64]*RedisNodeState
	Messages   map[string]Message
	Requests   []RedisRequest
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

func (r *RedisClusterState) CanDeliverRequest() bool {
	haveLeader := false
	for _, s := range r.NodeStates {
		if s.State == "leader" {
			haveLeader = true
			break
		}
	}
	return haveLeader
}

func (r *RedisClusterState) PendingRequests() []types.Request {
	out := make([]types.Request, 0)
	for _, r := range r.Requests {
		out = append(out, r.Copy())
	}
	return out
}

var _ types.PartitionedSystemState = &RedisClusterState{}

type RedisRaftEnv struct {
	clusterConfig *ClusterConfig
	network       *InterceptNetwork
	cluster       *Cluster

	curState *RedisClusterState

	stats map[string][]time.Duration
}

// For a given config, should only be instantiated once since it spins up a sever and binds the addr:port
func NewRedisRaftEnv(ctx context.Context, clusterConfig *ClusterConfig) *RedisRaftEnv {
	e := &RedisRaftEnv{
		clusterConfig: clusterConfig,
		network:       NewInterceptNetwork(ctx, clusterConfig.InterceptListenAddr),
		cluster:       nil,
		stats:         make(map[string][]time.Duration),
	}
	e.network.Start()

	// for duration analysis
	e.stats["start_times"] = make([]time.Duration, 0)
	e.stats["stop_times"] = make([]time.Duration, 0)

	return e
}

func (r *RedisRaftEnv) ReceiveRequest(req types.Request) types.PartitionedSystemState {
	newState := &RedisClusterState{
		NodeStates: make(map[uint64]*RedisNodeState),
		Requests:   make([]RedisRequest, 0),
		Messages:   r.network.GetAllMessages(),
	}
	for n, s := range r.curState.NodeStates {
		newState.NodeStates[n] = s.Copy()
	}

	haveLeader := false
	for _, s := range r.curState.NodeStates {
		if s.State == "leader" {
			haveLeader = true
			break
		}
	}

	remainingRequests := make([]RedisRequest, len(r.curState.Requests))
	for i, re := range r.curState.Requests {
		remainingRequests[i] = re.Copy()
	}
	if haveLeader {
		redisReq := req.(RedisRequest)
		if redisReq.Type == "Incr" {
			r.cluster.ExecuteAsync("INCR", "test")
		} else {
			r.cluster.ExecuteAsync("GET", "test")
		}
		remainingRequests = remainingRequests[1:]
	}

	for _, re := range remainingRequests {
		newState.Requests = append(newState.Requests, re.Copy())
	}

	return newState
}

func (r *RedisRaftEnv) DeliverMessages(messages []types.Message) types.PartitionedSystemState {
	wg := new(sync.WaitGroup)
	for _, m := range messages {
		rm, ok := m.(Message)
		if !ok {
			return r.curState
		}
		wg.Add(1)
		go func(rm Message, wg *sync.WaitGroup) {
			r.network.SendMessage(rm.ID)
			wg.Done()
		}(rm, wg)
	}
	wg.Wait()

	newState := &RedisClusterState{
		NodeStates: make(map[uint64]*RedisNodeState),
		Requests:   copyRequests(r.curState.Requests),
	}
	for id, s := range r.curState.NodeStates {
		newState.NodeStates[id] = s.Copy()
	}
	newState.Messages = r.network.GetAllMessages()
	r.curState = newState

	return newState
}

func (r *RedisRaftEnv) DropMessages(messages []types.Message) types.PartitionedSystemState {
	wg := new(sync.WaitGroup)
	for _, m := range messages {
		rm, ok := m.(Message)
		if !ok {
			return r.curState
		}
		wg.Add(1)
		go func(rm Message, wg *sync.WaitGroup) {
			r.network.DeleteMessage(rm.ID)
			wg.Done()
		}(rm, wg)
	}
	wg.Wait()
	newState := &RedisClusterState{
		NodeStates: make(map[uint64]*RedisNodeState),
		Requests:   copyRequests(r.curState.Requests),
	}
	for id, s := range r.curState.NodeStates {
		newState.NodeStates[id] = s.Copy()
	}
	newState.Messages = r.network.GetAllMessages()
	r.curState = newState

	return newState
}

func (r *RedisRaftEnv) Reset_old() types.PartitionedSystemState {
	if r.cluster != nil {
		r.cluster.Destroy()
	}
	r.network.Reset()
	r.cluster = NewCluster(r.clusterConfig)
	err := r.cluster.Start()
	if err != nil {
		panic(err)
	}

	// r.network.WaitForNodes(r.clusterConfig.NumNodes)

	r.network.WaitForNodes(r.clusterConfig.NumNodes)

	newState := &RedisClusterState{
		NodeStates: r.cluster.GetNodeStates(),
		Messages:   r.network.GetAllMessages(),
		Requests:   make([]RedisRequest, r.clusterConfig.NumRequests),
	}
	for i := 0; i < r.clusterConfig.NumRequests; i++ {
		req := RedisRequest{Type: "Incr"}
		if rand.Intn(2) == 1 {
			req.Type = "Get"
		}
		newState.Requests[i] = req.Copy()
	}

	r.curState = newState
	return newState
}

func (r *RedisRaftEnv) Reset() types.PartitionedSystemState {
	if r.cluster != nil {
		r.cluster.Destroy()
	}
	r.network.Reset()
	r.cluster = NewCluster(r.clusterConfig)

	// try to restart the cluster for a few times until it does not return an error
	trials := 0
	var err error
	for {
		err := r.cluster.Start()
		if err == nil || trials > 5 {
			break
		} else {
			if r.cluster != nil {
				r.cluster.Destroy()
			}
			r.network.Reset()
			r.cluster = NewCluster(r.clusterConfig)
			trials++
			time.Sleep(2 * time.Second)
		}
	}

	// err := r.cluster.Start()
	if err != nil {
		panic(err)
	}

	// r.network.WaitForNodes(r.clusterConfig.NumNodes)

	r.network.WaitForNodes(r.clusterConfig.NumNodes)

	newState := &RedisClusterState{
		NodeStates: r.cluster.GetNodeStates(),
		Messages:   r.network.GetAllMessages(),
		Requests:   make([]RedisRequest, r.clusterConfig.NumRequests),
	}
	for i := 0; i < r.clusterConfig.NumRequests; i++ {
		req := RedisRequest{Type: "Incr"}
		if rand.Intn(2) == 1 {
			req.Type = "Get"
		}
		newState.Requests[i] = req.Copy()
	}

	r.curState = newState
	return newState
}

func (r *RedisRaftEnv) Start(nodeID uint64) {
	if r.cluster == nil {
		return
	}
	node, ok := r.cluster.GetNode(int(nodeID))
	if !ok {
		return
	}
	start := time.Now()
	node.Start()
	dur := time.Since(start)

	r.stats["start_times"] = append(r.stats["start_times"], dur)
}

func (r *RedisRaftEnv) Stop(nodeID uint64) {
	if r.cluster == nil {
		return
	}
	node, ok := r.cluster.GetNode(int(nodeID))
	if !ok {
		return
	}
	start := time.Now()
	node.Stop()
	dur := time.Since(start)

	r.stats["stop_times"] = append(r.stats["stop_times"], dur)
}

func (r *RedisRaftEnv) Cleanup() {
	if r.cluster != nil {
		r.cluster.Destroy()
		r.cluster = nil
	}

	// sumStartTimes := time.Duration(0)
	// for _, d := range r.stats["start_times"] {
	// 	sumStartTimes += d
	// }
	// sumStopTimes := time.Duration(0)
	// for _, d := range r.stats["stop_times"] {
	// 	sumStopTimes += d
	// }
	// avgStartTime := time.Duration(int(sumStartTimes) / len(r.stats["start_times"]))
	// avgStopTime := time.Duration(int(sumStopTimes) / len(r.stats["stop_times"]))

	// fmt.Printf("Average start times: %s\n", avgStartTime.String())
	// fmt.Printf("Average stop times: %s\n", avgStopTime.String())
}

func (r *RedisRaftEnv) Tick() types.PartitionedSystemState {
	// time.Sleep(20 * time.Millisecond)
	time.Sleep(time.Duration(r.clusterConfig.TickLength) * time.Millisecond)
	newState := &RedisClusterState{
		NodeStates: r.cluster.GetNodeStates(),
		Messages:   r.network.GetAllMessages(),
		Requests:   copyRequests(r.curState.Requests),
	}
	r.curState = newState
	return newState
}

var _ types.PartitionedSystemEnvironment = &RedisRaftEnv{}
