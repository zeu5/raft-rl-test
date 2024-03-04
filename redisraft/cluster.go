package redisraft

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zeu5/raft-rl-test/types"
)

// An entry in a node's log
type RedisEntry struct {
	ID      string
	Term    int
	DataLen int
	Index   int
	Type    int
}

func (r RedisEntry) Copy() RedisEntry {
	return RedisEntry{
		ID:      r.ID,
		Term:    r.Term,
		DataLen: r.DataLen,
		Index:   r.Index,
		Type:    r.Type,
	}
}

func (r RedisEntry) String() string {
	return fmt.Sprintf("[ID:%s | Type:%s | I:%d | T:%d | Len:%d]", r.ID, EntryTypeToString(r.Type), r.Index, r.Term, r.DataLen)
}

func EntryTypeToString(val int) string {
	switch val {
	case 0:
		return "NORMAL"
	case 1:
		return "ADD_NONVOTING_NODE"
	case 2:
		return "ADD_NODE"
	case 3:
		return "NO_OP"
	default:
		return strconv.Itoa(val)
	}
}

// the state of a node
type RedisNodeState struct {
	Params    map[string]string
	Logs      []RedisEntry
	LogStdout string
	LogStderr string
	State     string
	Term      int
	Commit    int
	Index     int
	Applied   int
	Vote      int
	Lead      int
	Snapshot  int
}

func (r *RedisNodeState) Copy() *RedisNodeState {
	new := &RedisNodeState{
		Params:    make(map[string]string),
		Logs:      make([]RedisEntry, len(r.Logs)),
		LogStdout: r.LogStdout,
		LogStderr: r.LogStderr,
		State:     r.State,
		Term:      r.Term,
		Index:     r.Index,
		Commit:    r.Commit,
		Applied:   r.Applied,
		Vote:      r.Vote,
		Lead:      r.Lead,
		Snapshot:  r.Snapshot,
	}
	for k, v := range r.Params {
		new.Params[k] = v
	}
	for i, e := range r.Logs {
		new.Logs[i] = e.Copy()
	}
	return new
}

func EmptyRedisNodeState() *RedisNodeState {
	return &RedisNodeState{Params: make(map[string]string), Logs: make([]RedisEntry, 0)}
}

// configuration of a RedisNode
type RedisNodeConfig struct {
	ClusterID           int
	Port                int
	InterceptAddr       string
	InterceptListenAddr string
	NodeID              int
	WorkingDir          string
	RequestTimeout      int
	ElectionTimeout     int
	ModulePath          string
	BinaryPath          string
}

// RedisNode structure
type RedisNode struct {
	ID      int
	process *exec.Cmd
	client  *redis.Client
	config  *RedisNodeConfig
	ctx     context.Context
	cancel  context.CancelFunc

	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

func NewRedisNode(config *RedisNodeConfig) *RedisNode {
	portS := strconv.Itoa(config.Port)
	addr := "localhost:" + portS

	return &RedisNode{
		ID:      config.NodeID,
		process: nil,
		client: redis.NewClient(&redis.Options{
			Addr:        addr,
			DialTimeout: 10 * time.Millisecond,
		}),
		config: config,
		ctx:    nil,
		cancel: func() {},
		stdout: nil,
		stderr: nil,
	}
}

func (r *RedisNode) Create() {
	portS := strconv.Itoa(r.config.Port)
	addr := "localhost:" + portS

	periodicInterval := 100
	if r.config.RequestTimeout < periodicInterval {
		periodicInterval = int(r.config.RequestTimeout / 2)
	}

	serverArgs := []string{
		"--port", portS,
		"--bind", "0.0.0.0",
		"--dir", r.config.WorkingDir,
		"--dbfilename", fmt.Sprintf("redis%d.rdb", r.ID),
		"--loglevel", "debug",
		"--loadmodule", r.config.ModulePath,
		"--raft.addr", addr,
		"--raft.id", strconv.Itoa(r.ID),
		"--raft.log-filename", fmt.Sprintf("redis%d.db", r.ID),
		"--raft.use-test-network", "yes",
		"--raft.log-fsync", "no",
		"--raft.loglevel", "debug",
		"--raft.tls-enabled", "no",
		"--raft.trace", "off",
		"--raft.test-network-server-addr", r.config.InterceptAddr,
		"--raft.test-network-listen-addr", r.config.InterceptListenAddr,
		"--raft.periodic-interval", strconv.Itoa(periodicInterval),
		"--raft.request-timeout", strconv.Itoa(r.config.RequestTimeout),
		"--raft.election-timeout", strconv.Itoa(r.config.ElectionTimeout),
	}

	ctx, cancel := context.WithCancel(context.Background())
	r.process = exec.CommandContext(ctx, r.config.BinaryPath, serverArgs...)

	r.ctx = ctx
	r.cancel = cancel
	if r.stdout == nil {
		r.stdout = new(bytes.Buffer)
	}
	if r.stderr == nil {
		r.stderr = new(bytes.Buffer)
	}
	r.process.Stdout = r.stdout
	r.process.Stderr = r.stderr
}

// start the RedisNode server process, returns an error if already started
func (r *RedisNode) Start() error {
	if r.ctx != nil || r.process != nil {
		return errors.New("(redisraft:cluster.go:RedisNode:Start:1): redis server already started")
	}

	r.Create()
	return r.process.Start()
}

func (r *RedisNode) Cleanup() error {
	return os.RemoveAll(r.config.WorkingDir)
}

func (r *RedisNode) Stop() error {
	if r.ctx == nil || r.process == nil {
		// return fmt.Errorf(fmt.Sprintf("RedisNode[%d].Stop (redisraft:cluster.go:RedisNode:Stop:1): redis server not started", r.ID))
		// return nil
		r.ctx = nil
		r.cancel = func() {}
		r.process = nil

		return nil
	}
	select {
	case <-r.ctx.Done():
		// return fmt.Errorf(fmt.Sprintf("RedisNode[%d].Stop (redisraft:cluster.go:RedisNode:Stop:2): redis server already stopped", r.ID))
	default:
		r.cancel()
		err := r.process.Wait()

		if err != nil {
			if err.Error() != "signal: killed" {
				return fmt.Errorf(fmt.Sprintf("RedisNode[%d].Stop (redisraft:cluster.go:RedisNode:Stop:3): \n%s : %s", r.ID, err.Error(), r.process.ProcessState.String()))
			}
		}
	}

	r.ctx = nil
	r.cancel = func() {}
	r.process = nil

	return nil
}

func (r *RedisNode) Terminate() error {
	err := r.Stop()
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("RedisNode[%d].Terminate (redisraft:cluster.go:RedisNode:Terminate:1): \n%s", r.ID, err))
	}

	err = r.Cleanup()
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("RedisNode[%d].Terminate (redisraft:cluster.go:RedisNode:Terminate:2): \n%s", r.ID, err))
	}

	// stdout := strings.ToLower(r.stdout.String())
	// stderr := strings.ToLower(r.stderr.String())

	// if strings.Contains(stdout, "redis bug report") || strings.Contains(stderr, "redis bug report") {
	// 	return fmt.Errorf(fmt.Sprintf("RedisNode[%d].Terminate (redisraft:cluster.go:RedisNode:Terminate:3): failed to terminate, redis crashed", r.ID))
	// }
	return nil
}

func (r *RedisNode) TerminateCtx(epCtx *types.EpisodeContext) error {
	err := r.Stop()
	if err != nil {
		serr := strings.ToLower(r.stderr.String())
		epCtx.Report.AddLog(serr, fmt.Sprintf("redis_node_log_%d", r.ID))

		return fmt.Errorf(fmt.Sprintf("RedisNode[%d].Terminate (redisraft:cluster.go:RedisNode:Terminate:1): \n%s", r.ID, err))
	}

	err = r.Cleanup()
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("RedisNode[%d].Terminate (redisraft:cluster.go:RedisNode:Terminate:2): \n%s", r.ID, err))
	}

	// stdout := strings.ToLower(r.stdout.String())
	// stderr := strings.ToLower(r.stderr.String())

	// if strings.Contains(stdout, "redis bug report") || strings.Contains(stderr, "redis bug report") {
	// 	return fmt.Errorf(fmt.Sprintf("RedisNode[%d].Terminate (redisraft:cluster.go:RedisNode:Terminate:3): failed to terminate, redis crashed", r.ID))
	// }
	return nil
}

func (r *RedisNode) GetLogs() (string, string) {
	if r.stdout == nil || r.stderr == nil {
		return "", ""
	}
	return r.stdout.String(), r.stderr.String()
}

func (r *RedisNode) Execute(args ...string) error {
	if r.ctx == nil || r.process == nil {
		return errors.New("redis server not started")
	}
	select {
	case <-r.ctx.Done():
		return errors.New("redis server shutdown")
	default:
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	argsI := make([]interface{}, len(args))
	for i, a := range args {
		argsI[i] = a
	}
	_, err := r.client.Do(ctx, argsI...).Result()
	return err
}

func (r *RedisNode) ExecuteAsync(args ...string) error {
	if r.ctx == nil || r.process == nil {
		return errors.New("redis server not started")
	}
	select {
	case <-r.ctx.Done():
		return errors.New("redis server shutdown")
	default:
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		argsI := make([]interface{}, len(args))
		for i, a := range args {
			argsI[i] = a
		}
		r.client.Do(ctx, argsI...).Result()
	}()
	return nil
}

func (r *RedisNode) cluster(args ...string) error {
	cmdArgs := []string{"RAFT.CLUSTER"}
	cmdArgs = append(cmdArgs, args...)

	return r.Execute(cmdArgs...)
}

func (r *RedisNode) Info() (*RedisNodeState, error) {
	if r.ctx == nil || r.process == nil {
		return nil, errors.New("process stopped")
	}

	// TODO: is the timeout duration sufficient?
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	info, err := r.client.Info(ctx, "raft").Result()
	if err != nil {
		return nil, err
	}
	pInfo := ParseInfo(info)
	stdout, stderr := r.GetLogs()
	pInfo.LogStdout = stdout
	pInfo.LogStderr = stderr
	return pInfo, nil
}

type ClusterBaseConfig struct {
	NumNodes              int    // number of nodes in the cluster
	RaftModulePath        string // ???
	RedisServerBinaryPath string // ???

	BasePort            int
	BaseInterceptPort   int
	ID                  int
	InterceptListenPort int

	LogLevel        string
	RequestTimeout  int // heartbeat timeout in milliseconds
	ElectionTimeout int // new election timeout in milliseconds
	NumRequests     int // number of client requests available
	TickLength      int // length of a tick in milliseconds
}

func (c *ClusterBaseConfig) Instantiate(parallelIndex int) *ClusterConfig {
	return &ClusterConfig{
		NumNodes:              c.NumNodes,
		RaftModulePath:        c.RaftModulePath,
		RedisServerBinaryPath: c.RedisServerBinaryPath,
		BasePort:              c.BasePort,
		BaseInterceptPort:     c.BaseInterceptPort,
		ID:                    c.ID,
		InterceptListenAddr:   fmt.Sprintf("localhost:%d", c.InterceptListenPort+parallelIndex),
		WorkingDir:            fmt.Sprintf("/tmp/redisraft_%d_%d", c.ID, parallelIndex),
		LogLevel:              c.LogLevel,
		RequestTimeout:        c.RequestTimeout,
		ElectionTimeout:       c.ElectionTimeout,
		NumRequests:           c.NumRequests,
		TickLength:            c.TickLength,
	}
}

type ClusterConfig struct {
	NumNodes              int    // number of nodes in the cluster
	RaftModulePath        string // ???
	RedisServerBinaryPath string // ???
	BasePort              int
	BaseInterceptPort     int
	ID                    int
	InterceptListenAddr   string
	WorkingDir            string
	LogLevel              string
	RequestTimeout        int // heartbeat timeout in milliseconds
	ElectionTimeout       int // new election timeout in milliseconds
	NumRequests           int // number of client requests available
	TickLength            int // length of a tick in milliseconds
}

func (r *ClusterConfig) Printable() string {
	result := "RedisClusterConfig: \n"
	result = fmt.Sprintf("%s Replicas: %d\n", result, r.NumNodes)
	result = fmt.Sprintf("%s ElectionTick: %d ms\n", result, r.ElectionTimeout)
	result = fmt.Sprintf("%s HeartbeatTick: %d ms\n", result, r.RequestTimeout)
	result = fmt.Sprintf("%s TickLength: %d ms\n", result, r.TickLength)
	result = fmt.Sprintf("%s Requests: %d\n", result, r.NumRequests)
	return result
}

func (c *ClusterConfig) Copy() *ClusterConfig {
	return &ClusterConfig{
		NumNodes:              c.NumNodes,
		RaftModulePath:        c.RaftModulePath,
		RedisServerBinaryPath: c.RedisServerBinaryPath,
		BasePort:              c.BasePort,
		BaseInterceptPort:     c.BaseInterceptPort,
		ID:                    c.ID,
		InterceptListenAddr:   c.InterceptListenAddr,
		WorkingDir:            c.WorkingDir,
		LogLevel:              c.LogLevel,
		RequestTimeout:        c.RequestTimeout,
		ElectionTimeout:       c.ElectionTimeout,
		NumRequests:           c.NumRequests,
		TickLength:            c.TickLength,
	}
}

func (c *ClusterConfig) SetDefaults() {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := path.Dir(ex)
	if c.RaftModulePath == "" {
		c.RaftModulePath = path.Join(exPath, "/redisraft/redisraft.so")
		// "/home/snagendra/Fuzzing/redisraft-fuzzing/redisraft.so"
	}
	if c.RedisServerBinaryPath == "" {
		c.RedisServerBinaryPath = path.Join(exPath, "/redisraft/redis-server")
		// "/home/snagendra/Fuzzing/redis/src/redis-server"
	}
	if c.WorkingDir == "" {
		c.WorkingDir = "/home/snagendra/Fuzzing/redisraft-fuzzing/tests/tmp"
	}
	if _, err := os.Stat(c.WorkingDir); err == nil {
		os.RemoveAll(c.WorkingDir)
	}
	os.MkdirAll(c.WorkingDir, 0777)

	if c.LogLevel == "" {
		c.LogLevel = "INFO"
	}
	if c.RequestTimeout == 0 {
		c.RequestTimeout = 40
	}
	if c.ElectionTimeout == 0 {
		c.ElectionTimeout = 200
	}
	if c.TickLength == 0 {
		c.TickLength = 40
	}
}

func (c *ClusterConfig) GetNodeConfig(id int) *RedisNodeConfig {
	nodeWorkDir := path.Join(c.WorkingDir, strconv.Itoa(id))
	if _, err := os.Stat(nodeWorkDir); err == nil {
		os.RemoveAll(nodeWorkDir)
	}
	os.MkdirAll(nodeWorkDir, 0777)

	return &RedisNodeConfig{
		ClusterID:           c.ID,
		Port:                c.BasePort + id,
		InterceptAddr:       c.InterceptListenAddr,
		InterceptListenAddr: fmt.Sprintf("localhost:%d", c.BaseInterceptPort+id),
		NodeID:              id,
		WorkingDir:          nodeWorkDir,
		RequestTimeout:      c.RequestTimeout,
		ElectionTimeout:     c.ElectionTimeout,
		ModulePath:          c.RaftModulePath,
		BinaryPath:          c.RedisServerBinaryPath,
	}
}

type Cluster struct {
	Nodes  map[int]*RedisNode
	config *ClusterConfig
}

func NewCluster(config *ClusterConfig) *Cluster {
	config.SetDefaults()
	c := &Cluster{
		config: config,
		Nodes:  make(map[int]*RedisNode),
	}
	// Make nodes
	for i := 1; i <= c.config.NumNodes; i++ {
		nConfig := config.GetNodeConfig(i)
		c.Nodes[i] = NewRedisNode(nConfig)
	}

	return c
}

func (c *Cluster) GetNodeStates() map[uint64]*RedisNodeState {
	out := make(map[uint64]*RedisNodeState)
	outLock := new(sync.Mutex)
	wg := new(sync.WaitGroup)
	for id, node := range c.Nodes {
		wg.Add(1)
		go func(node *RedisNode, id int) {
			state, err := node.Info()
			if err != nil {
				e := EmptyRedisNodeState()
				e.LogStdout, e.LogStderr = node.GetLogs()
				outLock.Lock()
				out[uint64(id)] = e
				outLock.Unlock()
			} else {
				outLock.Lock()
				out[uint64(id)] = state.Copy()
				outLock.Unlock()
			}
			wg.Done()
		}(node, id)
	}
	wg.Wait()
	return out
}

// returns the updated map of RedisNodeState - version returning also the execution time in microseconds
func (c *Cluster) GetNodeStates_Time() (map[uint64]*RedisNodeState, int64) {
	timeStart := time.Now().UnixMicro()
	out := make(map[uint64]*RedisNodeState)
	for id, node := range c.Nodes {
		state, err := node.Info()
		if err != nil {
			e := EmptyRedisNodeState()
			e.LogStdout, e.LogStderr = node.GetLogs()
			out[uint64(id)] = e
			continue
		}
		out[uint64(id)] = state.Copy()
	}
	timeEnd := time.Now().UnixMicro()

	return out, timeEnd - timeStart
}

func (c *Cluster) Start() error {
	primaryAddr := ""
	for i := 1; i <= c.config.NumNodes; i++ {
		node := c.Nodes[i]
		if err := node.Start(); err != nil {
			return fmt.Errorf("error starting node %d (redisraft:cluster.go:Cluster:Start:1):\n%s", i, err)
		}
		if i == 1 {
			if err := node.cluster("init"); err != nil {
				return fmt.Errorf("failed to initialize (redisraft:cluster.go:Cluster:Start:2):\n%s", err)
			}
			primaryAddr = "localhost:" + strconv.Itoa(node.config.Port)
		} else {
			if err := node.cluster("join", primaryAddr); err != nil {
				return fmt.Errorf("failed to join cluster (redisraft:cluster.go:Cluster:Start:3):\n%s", err)
			}
		}
	}
	return nil
}

func (c *Cluster) Destroy() error {
	var err error = nil
	for i, node := range c.Nodes {
		err = node.Terminate()
		if err != nil {
			return fmt.Errorf("failed to terminate node %d (redisraft:cluster.go:Cluster:Destroy:1):\n%s", i, err)
		}
	}
	return err
}

func (c *Cluster) DestroyCtx(epCtx *types.EpisodeContext) error {
	var err error = nil
	for i, node := range c.Nodes {
		err = node.TerminateCtx(epCtx)
		if err != nil {
			return fmt.Errorf("failed to terminate node %d (redisraft:cluster.go:Cluster:Destroy:1):\n%s", i, err)
		}
	}

	time.Sleep(1 * time.Second)

	return err
}

func (c *Cluster) GetNode(id int) (*RedisNode, bool) {
	node, ok := c.Nodes[id]
	return node, ok
}

func (c *Cluster) GetLogs() string {
	logLines := []string{}
	for nodeID, node := range c.Nodes {
		logLines = append(logLines, fmt.Sprintf("logs for node: %d\n", nodeID))
		stdout, stderr := node.GetLogs()
		logLines = append(logLines, "----- Stdout -----", stdout, "----- Stderr -----", stderr, "\n\n")
	}
	return strings.Join(logLines, "\n")
}

func (c *Cluster) Execute(id int, args ...string) error {
	node := c.Nodes[id]
	return node.Execute(args...)
}

func (c *Cluster) ExecuteAsync(id int, args ...string) error {
	node := c.Nodes[id]
	return node.ExecuteAsync(args...)
}
