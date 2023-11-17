package cbft

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

type ProposerInfo struct {
	Address []byte
	Index   int
}

func (p *ProposerInfo) Copy() *ProposerInfo {
	out := &ProposerInfo{
		Index: p.Index,
	}
	if p.Address != nil {
		out.Address = make([]byte, len(p.Address))
		copy(out.Address, p.Address)
	}
	return out
}

type CometNodeState struct {
	HeightRoundStep   string        `json:"height/round/step"`
	ProposalBlockHash []byte        `json:"proposal_block_hash"`
	LockedBlockHash   []byte        `json:"locked_block_hash"`
	ValidBlockHash    []byte        `json:"valid_block_hash"`
	Votes             []byte        `json:"height_vote_set"`
	Proposer          *ProposerInfo `json:"proposer"`

	LogStdout string `json:"-"`
	LogStderr string `json:"-"`
}

func EmptyCometNodeState() *CometNodeState {
	return &CometNodeState{}
}

func (r *CometNodeState) Copy() *CometNodeState {
	out := &CometNodeState{
		HeightRoundStep: r.HeightRoundStep,
		LogStdout:       r.LogStdout,
		LogStderr:       r.LogStderr,
	}
	if r.ProposalBlockHash != nil {
		out.ProposalBlockHash = make([]byte, len(r.ProposalBlockHash))
		copy(out.ProposalBlockHash, r.ProposalBlockHash)
	}
	if r.LockedBlockHash != nil {
		out.LockedBlockHash = make([]byte, len(r.LockedBlockHash))
		copy(out.LockedBlockHash, r.LockedBlockHash)
	}
	if r.ValidBlockHash != nil {
		out.ValidBlockHash = make([]byte, len(r.ValidBlockHash))
		copy(out.ValidBlockHash, r.ValidBlockHash)
	}
	if r.Votes != nil {
		out.Votes = make([]byte, len(r.Votes))
		copy(out.Votes, r.Votes)
	}
	if r.Proposer != nil {
		out.Proposer = r.Proposer.Copy()
	}
	return out
}

type CometNodeConfig struct {
	ID         int
	RPCAddress string
	BinaryPath string
	WorkingDir string
}

type CometNode struct {
	ID      int
	process *exec.Cmd
	config  *CometNodeConfig
	ctx     context.Context
	cancel  context.CancelFunc

	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

func NewCometNode(config *CometNodeConfig) *CometNode {
	return &CometNode{
		ID:     config.ID,
		config: config,
	}
}

func (r *CometNode) Create() {
	serverArgs := []string{
		"-home", r.config.WorkingDir,
		"istart",
	}

	ctx, cancel := context.WithCancel(context.Background())
	r.process = exec.CommandContext(ctx, r.config.BinaryPath, serverArgs...)
	r.process.Dir = r.config.WorkingDir

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

func (r *CometNode) Start() error {
	if r.ctx != nil || r.process != nil {
		return errors.New("comet server already started")
	}

	r.Create()
	return r.process.Start()
}

func (r *CometNode) Cleanup() {
	os.RemoveAll(r.config.WorkingDir)
}

func (r *CometNode) Stop() error {
	if r.ctx == nil || r.process == nil {
		return errors.New("comet server not started")
	}
	select {
	case <-r.ctx.Done():
		return errors.New("comet server already stopped")
	default:
	}
	r.cancel()
	err := r.process.Wait()
	r.ctx = nil
	r.cancel = func() {}
	r.process = nil

	return err
}

func (r *CometNode) Terminate() error {
	r.Stop()
	r.Cleanup()

	// stdout := strings.ToLower(r.stdout.String())
	// stderr := strings.ToLower(r.stderr.String())

	// Todo: check for bugs in the logs
	return nil
}

func (r *CometNode) GetLogs() (string, string) {
	if r.stdout == nil || r.stderr == nil {
		return "", ""
	}
	return r.stdout.String(), r.stderr.String()
}

func (r *CometNode) Info() (*CometNodeState, error) {
	// Todo: figure out how to get the node state
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 5 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DisableKeepAlives:     true,
		},
	}
	resp, err := client.Get("http://" + r.config.RPCAddress + "/consensus_state")
	if err != nil {
		return nil, fmt.Errorf("failed to read state from node: %s", err)
	}
	defer resp.Body.Close()

	stateBS, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read state from node: %s", err)
	}

	newState := &CometNodeState{}
	err = json.Unmarshal(stateBS, newState)
	if err != nil {
		return nil, fmt.Errorf("failed to read state from node: %s", err)
	}
	newState.LogStdout = r.stdout.String()
	newState.LogStderr = r.stderr.String()

	return newState, nil
}

type CometClusterConfig struct {
	NumNodes            int
	CometBinaryPath     string
	InterceptListenPort int
	BaseRPCPort         int
	BaseWorkingDir      string
}

func (c *CometClusterConfig) GetNodeConfig(id int) *CometNodeConfig {
	return &CometNodeConfig{
		ID:         id,
		WorkingDir: path.Join(c.BaseWorkingDir, fmt.Sprintf("node%d", id)),
		RPCAddress: fmt.Sprintf("localhost:%d", c.BaseRPCPort+(id-1)),
		BinaryPath: c.CometBinaryPath,
	}
}

type CometCluster struct {
	Nodes  map[int]*CometNode
	config *CometClusterConfig
}

func NewCluster(config *CometClusterConfig) *CometCluster {
	c := &CometCluster{
		config: config,
		Nodes:  make(map[int]*CometNode),
	}
	// Make nodes
	for i := 1; i <= c.config.NumNodes; i++ {
		nConfig := config.GetNodeConfig(i)
		c.Nodes[i] = NewCometNode(nConfig)
	}

	return c
}

func (c *CometCluster) GetNodeStates() map[uint64]*CometNodeState {
	out := make(map[uint64]*CometNodeState)
	for id, node := range c.Nodes {
		state, err := node.Info()
		if err != nil {
			out[uint64(id)] = EmptyCometNodeState()
			continue
		}
		out[uint64(id)] = state.Copy()
	}
	return out
}

func (c *CometCluster) Start() error {
	for i := 1; i <= c.config.NumNodes; i++ {
		node := c.Nodes[i]
		if err := node.Start(); err != nil {
			return fmt.Errorf("error starting node %d: %s", i, err)
		}
	}
	return nil
}

func (c *CometCluster) Destroy() error {
	var err error = nil
	for _, node := range c.Nodes {
		err = node.Terminate()
	}
	return err
}

func (c *CometCluster) GetNode(id int) (*CometNode, bool) {
	node, ok := c.Nodes[id]
	return node, ok
}

func (c *CometCluster) GetLogs() string {
	logLines := []string{}
	for nodeID, node := range c.Nodes {
		logLines = append(logLines, fmt.Sprintf("logs for node: %d\n", nodeID))
		stdout, stderr := node.GetLogs()
		logLines = append(logLines, "----- Stdout -----", stdout, "----- Stderr -----", stderr, "\n\n")
	}
	return strings.Join(logLines, "\n")
}
