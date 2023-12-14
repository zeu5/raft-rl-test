package redisraft

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path"
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

// return the replica state for the specified ID
func (r *RedisClusterState) GetReplicaState(id uint64) types.ReplicaState {
	s := r.NodeStates[id]
	return s
}

// return a copy of the pending messages in the cluster
func (r *RedisClusterState) PendingMessages() map[string]types.Message {
	out := make(map[string]types.Message)
	for k, m := range r.Messages {
		out[k] = m
	}
	return out
}

// return true if there is a replica in state leader
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

// return a copy of the pending requests
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

	timeStats map[string][]time.Duration
	intStats  map[string][]int

	// record duration between ticks
	timeTickDurations [][]int64 // map[episode] = [tick gaps in microseconds]
	timeLastRecorded  int64     //
	timeEpIndex       int
	savePath          string
	printStats        bool // if true, store stats to files -- needs to be set with the setter method
}

// For a given config, should only be instantiated once since it spins up a sever and binds the addr:port
func NewRedisRaftEnv(ctx context.Context, clusterConfig *ClusterConfig, savePath ...string) *RedisRaftEnv {
	e := &RedisRaftEnv{
		clusterConfig:     clusterConfig,
		network:           NewInterceptNetwork(ctx, clusterConfig.InterceptListenAddr),
		cluster:           nil,
		timeStats:         make(map[string][]time.Duration),
		intStats:          make(map[string][]int),
		timeTickDurations: make([][]int64, 0),
		timeEpIndex:       0,
		savePath:          savePath[0],
		printStats:        false, // change calling SetPrintStats()
	}
	e.network.Start()

	// for duration analysis
	e.timeStats["start_times"] = make([]time.Duration, 0)
	e.timeStats["stop_times"] = make([]time.Duration, 0)

	e.timeStats["GetNodeStates"] = make([]time.Duration, 0)
	e.timeStats["DeliverMessages"] = make([]time.Duration, 0)
	e.timeStats["GetAllMessages"] = make([]time.Duration, 0)
	e.timeStats["DropMessages"] = make([]time.Duration, 0)

	e.intStats["NumberOfMessages"] = make([]int, 0)

	// for tick length analysis
	if len(savePath) == 1 {
		if _, ok := os.Stat(savePath[0]); ok == nil {
			os.RemoveAll(savePath[0])
		}
		os.MkdirAll(savePath[0], 0777)
	}
	e.timeTickDurations = append(e.timeTickDurations, make([]int64, 0)) // initialize with the first episode list

	return e
}

// change the value of printStats, true: prints to file the episode time stats
func (r *RedisRaftEnv) SetPrintStats(value bool) {
	r.printStats = value
}

// deliver a request to leader action
func (r *RedisRaftEnv) ReceiveRequest(req types.Request) types.PartitionedSystemState {
	// fmt.Print("ReceiveRequest function") // DEBUG
	newState := &RedisClusterState{
		NodeStates: make(map[uint64]*RedisNodeState),
		Requests:   make([]RedisRequest, 0),
		Messages:   r.network.GetAllMessages(),
	}
	for n, s := range r.curState.NodeStates {
		newState.NodeStates[n] = s.Copy()
	}

	haveLeader := false
	leaderId := 0
	for id, s := range r.curState.NodeStates {
		if s.State == "leader" {
			haveLeader = true
			leaderId = int(id)
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
			r.cluster.ExecuteAsync(leaderId, "INCR", "test")
		} else {
			r.cluster.ExecuteAsync(leaderId, "GET", "test")
		}
		remainingRequests = remainingRequests[1:]
	}

	for _, re := range remainingRequests {
		newState.Requests = append(newState.Requests, re.Copy())
	}

	start := time.Now() // time stats

	newState.NodeStates = r.cluster.GetNodeStates()

	dur := time.Since(start)                                                 // time stats
	r.timeStats["GetNodeStates"] = append(r.timeStats["GetNodeStates"], dur) // time stats

	r.curState = newState
	return newState
}

func (r *RedisRaftEnv) DeliverMessages(messages []types.Message) types.PartitionedSystemState {
	// fmt.Print("DeliverMessages function") // DEBUG
	start := time.Now() // time stats

	wg := new(sync.WaitGroup)    // create WaitGroup
	for _, m := range messages { // foreach message
		rm, ok := m.(Message) // cast into redis message
		if !ok {              // if fails return current state?
			return r.curState
		}
		wg.Add(1) // increase waitgroup counter

		// routine calling the send message and decreasing the counter upon completion
		go func(rm Message, wg *sync.WaitGroup) {
			r.network.SendMessage(rm.ID)
			wg.Done()
		}(rm, wg)
	}
	wg.Wait()                                                                    // wait for all routines to return
	dur := time.Since(start)                                                     // time stats
	r.timeStats["DeliverMessages"] = append(r.timeStats["DeliverMessages"], dur) // time stats

	newState := &RedisClusterState{
		NodeStates: make(map[uint64]*RedisNodeState),
		Requests:   copyRequests(r.curState.Requests),
	}
	for id, s := range r.curState.NodeStates {
		newState.NodeStates[id] = s.Copy()
	}
	start2 := time.Now() // time stats

	newState.Messages = r.network.GetAllMessages()
	r.intStats["NumberOfMessages"] = append(r.intStats["NumberOfMessages"], len(messages)) // int stats

	dur2 := time.Since(start2)                                                  // time stats
	r.timeStats["GetAllMessages"] = append(r.timeStats["GetAllMessages"], dur2) // time stats

	r.curState = newState

	return newState
}

func (r *RedisRaftEnv) DropMessages(messages []types.Message) types.PartitionedSystemState {
	// fmt.Print("DropMessages function") // DEBUG
	start := time.Now() // time stats
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

	start2 := time.Now() // time stats

	newState.Messages = r.network.GetAllMessages()
	r.intStats["NumberOfMessages"] = append(r.intStats["NumberOfMessages"], len(messages)) // int stats

	dur2 := time.Since(start2)                                                  // time stats
	r.timeStats["GetAllMessages"] = append(r.timeStats["GetAllMessages"], dur2) // time stats

	r.curState = newState

	dur := time.Since(start)                                               // time stats
	r.timeStats["DropMessages"] = append(r.timeStats["DropMessages"], dur) // time stats

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
	// fmt.Print("Reset function") // DEBUG
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

	r.network.WaitForNodes(r.clusterConfig.NumNodes)

	newState := &RedisClusterState{
		NodeStates: r.cluster.GetNodeStates(),
		Messages:   r.network.GetAllMessages(),
		Requests:   make([]RedisRequest, r.clusterConfig.NumRequests),
	}
	for i := 0; i < r.clusterConfig.NumRequests; i++ {
		req := RedisRequest{Type: "Incr"}
		// if rand.Intn(2) == 1 {
		// 	req.Type = "Get"
		// }
		newState.Requests[i] = req.Copy()
	}

	r.curState = newState

	if len(r.timeTickDurations[r.timeEpIndex]) > 0 { // time stats
		// fmt.Print("printing one episode\n") // DEBUG
		if r.printStats {
			fileName := fmt.Sprintf("ticks_episode_%d", r.timeEpIndex) // start counting episodes from zero
			filePath := path.Join(r.savePath, fileName)
			text := ""
			text = fmt.Sprintf("%s Number of ticks: %d\n", text, len(r.timeTickDurations[r.timeEpIndex]))
			for i := 0; i < len(r.timeTickDurations[r.timeEpIndex]); i++ {
				text = fmt.Sprintf("%s %d", text, r.timeTickDurations[r.timeEpIndex][i])
			}
			// fmt.Print(text) // DEBUG
			types.WriteToFile(filePath, text)

			fileName = fmt.Sprintf("timeStats_episode_%d", r.timeEpIndex) // start counting episodes from zero
			filePath = path.Join(r.savePath, fileName)
			text = PrintableTimeStats(r.timeStats)
			text = text + "\n" + PrintableIntStats(r.intStats)
			types.WriteToFile(filePath, text)

			// set up new list and index
			r.timeTickDurations = append(r.timeTickDurations, make([]int64, 0)) // add a new episode as a list
			r.timeEpIndex = len(r.timeTickDurations) - 1                        // set the current episode index as the last entry in the list of lists
		} else { // just reset ticks list
			r.timeTickDurations[r.timeEpIndex] = make([]int64, 0)
		}
		r.ResetStats() // reset stats lists for the new episode
	}

	// for tick length recording
	// fmt.Print("update list") // DEBUG

	r.timeLastRecorded = time.Now().UnixMicro() // store current system time

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

	r.timeStats["start_times"] = append(r.timeStats["start_times"], dur)
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

	r.timeStats["stop_times"] = append(r.timeStats["stop_times"], dur)
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
	// fmt.Print("tick function") // DEBUG
	// time.Sleep(20 * time.Millisecond)
	time.Sleep(time.Duration(r.clusterConfig.TickLength) * time.Millisecond)

	timeCurrent := time.Now().UnixMicro()                                                           // read current time
	timeDifference := timeCurrent - r.timeLastRecorded                                              // compute difference with previously stored
	r.timeLastRecorded = timeCurrent                                                                // store new time
	r.timeTickDurations[r.timeEpIndex] = append(r.timeTickDurations[r.timeEpIndex], timeDifference) // add an entry in the episode with the recorded time gap

	start := time.Now()

	nStates := r.cluster.GetNodeStates()

	dur := time.Since(start)                                                 // time stats
	r.timeStats["GetNodeStates"] = append(r.timeStats["GetNodeStates"], dur) // time stats

	start = time.Now()

	messages := r.network.GetAllMessages()
	r.intStats["NumberOfMessages"] = append(r.intStats["NumberOfMessages"], len(messages)) // int stats

	dur = time.Since(start)                                                    // time stats
	r.timeStats["GetAllMessages"] = append(r.timeStats["GetAllMessages"], dur) // time stats

	newState := &RedisClusterState{
		NodeStates: nStates,
		Messages:   messages,
		Requests:   copyRequests(r.curState.Requests),
	}
	r.curState = newState
	return newState
}

// reset time stats lists
func (r *RedisRaftEnv) ResetStats() {
	r.timeStats["start_times"] = make([]time.Duration, 0)
	r.timeStats["stop_times"] = make([]time.Duration, 0)

	r.timeStats["GetNodeStates"] = make([]time.Duration, 0)
	r.timeStats["DeliverMessages"] = make([]time.Duration, 0)
	r.timeStats["GetAllMessages"] = make([]time.Duration, 0)
	r.timeStats["DropMessages"] = make([]time.Duration, 0)

	r.intStats["NumberOfMessages"] = make([]int, 0)
}

func PrintableTimeStats(data map[string]([]time.Duration)) string {
	result := ""

	for label, durations := range data {
		result = fmt.Sprintf("%s%s (count %d):\n", result, label, len(durations))
		for _, dur := range durations {
			result = fmt.Sprintf("%s %d", result, dur.Milliseconds())
		}
		result = result + "\n"
	}

	return result
}

func PrintableIntStats(data map[string]([]int)) string {
	result := ""

	for label, values := range data {
		result = fmt.Sprintf("%s%s (count %d):\n", result, label, len(values))
		for _, val := range values {
			result = fmt.Sprintf("%s %d", result, val)
		}
		result = result + "\n"
	}

	return result
}

var _ types.PartitionedSystemEnvironment = &RedisRaftEnv{}
