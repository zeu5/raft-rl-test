package ratis

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type RatisNodeState struct {
	State   string
	Term    int
	Commit  int
	Applied int
	Vote    int
	Lead    int
}

func EmptyRatisNodeState() *RatisNodeState {
	return &RatisNodeState{}
}

func (r *RatisNodeState) Copy() *RatisNodeState {
	return &RatisNodeState{
		State:   r.State,
		Term:    r.Term,
		Commit:  r.Commit,
		Applied: r.Applied,
		Vote:    r.Vote,
		Lead:    r.Lead,
	}
}

type RatisNodeConfig struct {
	ID                  int
	JarPath             string
	InterceptPort       int
	InterceptListenPort int
	Peers               string
	GroupID             string
	WorkingDir          string
}

type RatisNode struct {
	ID      int
	process *exec.Cmd
	config  *RatisNodeConfig
	ctx     context.Context
	cancel  context.CancelFunc

	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

func NewRatisNode(config *RatisNodeConfig) *RatisNode {
	return &RatisNode{
		ID:     config.ID,
		config: config,
	}
}

func (r *RatisNode) Create() {
	// Todo: add command arguments here to instantiate the ratis node
	serverArgs := []string{
		"-cp",
		r.config.JarPath,
		"org.apache.ratis.examples.counter.server.CounterServer",
		strconv.Itoa(r.config.ID),
		strconv.Itoa(r.config.InterceptPort),
		strconv.Itoa(r.config.InterceptListenPort),
		strconv.Itoa(r.config.ID),
		r.config.Peers,
		r.config.GroupID,
	}

	ctx, cancel := context.WithCancel(context.Background())
	r.process = exec.CommandContext(ctx, "java", serverArgs...)
	r.process.Dir = r.config.WorkingDir

	r.ctx = ctx
	r.cancel = cancel
	r.stdout = new(bytes.Buffer)
	r.stderr = new(bytes.Buffer)
	r.process.Stdout = r.stdout
	r.process.Stderr = r.stderr
}

func (r *RatisNode) Start() error {
	if r.ctx != nil || r.process != nil {
		return errors.New("ratis server already started")
	}

	r.Create()
	return r.process.Start()
}

func (r *RatisNode) Cleanup() {
	// Todo: figure out what to do to cleanup
}

func (r *RatisNode) Stop() error {
	if r.ctx == nil || r.process == nil {
		return errors.New("ratis server not started")
	}
	select {
	case <-r.ctx.Done():
		return errors.New("ratis server already stopped")
	default:
	}
	r.cancel()
	err := r.process.Wait()
	r.ctx = nil
	r.cancel = func() {}
	r.process = nil

	return err
}

func (r *RatisNode) Terminate() error {
	r.Stop()
	r.Cleanup()

	// stdout := strings.ToLower(r.stdout.String())
	// stderr := strings.ToLower(r.stderr.String())

	// Todo: check for bugs in the logs
	return nil
}

func (r *RatisNode) GetLogs() (string, string) {
	if r.stdout == nil || r.stderr == nil {
		return "", ""
	}
	return r.stdout.String(), r.stderr.String()
}

func (r *RatisNode) Info() (*RatisNodeState, error) {
	// Todo: figure out how to get the node state
	return nil, errors.New("not implemented")
}

type RatisClusterConfig struct {
	NumNodes            int
	RatisJarPath        string
	BasePort            int
	BaseInterceptPort   int
	InterceptListenPort int
	GroupID             string
	WorkingDir          string
}

func (c *RatisClusterConfig) GetNodeConfig(id int) *RatisNodeConfig {
	peers := make([]string, 0)
	for i := 1; i < c.NumNodes+1; i++ {
		peers = append(peers, fmt.Sprintf("localhost:%d", c.BasePort+i))
	}

	return &RatisNodeConfig{
		ID:                  id,
		InterceptPort:       c.InterceptListenPort,
		InterceptListenPort: c.BaseInterceptPort + id,
		JarPath:             c.RatisJarPath,
		Peers:               strings.Join(peers, ","),
		GroupID:             c.GroupID,
		WorkingDir:          c.WorkingDir,
	}
}

type RatisCluster struct {
	Nodes  map[int]*RatisNode
	config *RatisClusterConfig
}

func NewCluster(config *RatisClusterConfig) *RatisCluster {
	c := &RatisCluster{
		config: config,
		Nodes:  make(map[int]*RatisNode),
	}
	// Make nodes
	for i := 1; i <= c.config.NumNodes; i++ {
		nConfig := config.GetNodeConfig(i)
		c.Nodes[i] = NewRatisNode(nConfig)
	}

	return c
}

func (c *RatisCluster) GetNodeStates() map[uint64]*RatisNodeState {
	out := make(map[uint64]*RatisNodeState)
	for id, node := range c.Nodes {
		state, err := node.Info()
		if err != nil {
			out[uint64(id)] = EmptyRatisNodeState()
			continue
		}
		out[uint64(id)] = state.Copy()
	}
	return out
}

func (c *RatisCluster) Start() error {
	for i := 1; i <= c.config.NumNodes; i++ {
		node := c.Nodes[i]
		if err := node.Start(); err != nil {
			return fmt.Errorf("error starting node %d: %s", i, err)
		}
	}
	return nil
}

func (c *RatisCluster) Destroy() error {
	var err error = nil
	for _, node := range c.Nodes {
		err = node.Terminate()
	}
	return err
}

func (c *RatisCluster) GetNode(id int) (*RatisNode, bool) {
	node, ok := c.Nodes[id]
	return node, ok
}

func (c *RatisCluster) GetLogs() string {
	logLines := []string{}
	for nodeID, node := range c.Nodes {
		logLines = append(logLines, fmt.Sprintf("logs for node: %d\n", nodeID))
		stdout, stderr := node.GetLogs()
		logLines = append(logLines, "----- Stdout -----", stdout, "----- Stderr -----", stderr, "\n\n")
	}
	return strings.Join(logLines, "\n")
}
