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
	"strconv"
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
	Height            int               `json:"-"`
	Round             int               `json:"-"`
	Step              string            `json:"-"`
	HeightRoundStep   string            `json:"height/round/step"`
	ProposalBlockHash string            `json:"proposal_block_hash"`
	LockedBlockHash   string            `json:"locked_block_hash"`
	ValidBlockHash    string            `json:"valid_block_hash"`
	Votes             []*CometNodeVotes `json:"height_vote_set"`
	Proposer          *ProposerInfo     `json:"proposer"`

	LogStdout string `json:"-"`
	LogStderr string `json:"-"`
}

type CometNodeVotes struct {
	Round              int      `json:"round"`
	Prevotes           []string `json:"prevotes"`
	PrevotesBitArray   string   `json:"prevotes_bit_array"`
	Precommits         []string `json:"precommits"`
	PrecommitsBitArray string   `json:"precommits_bit_array"`
}

func (c *CometNodeVotes) Copy() *CometNodeVotes {
	out := &CometNodeVotes{
		Round:              c.Round,
		PrevotesBitArray:   c.PrevotesBitArray,
		PrecommitsBitArray: c.PrecommitsBitArray,
	}
	if c.Precommits != nil {
		out.Precommits = make([]string, len(c.Precommits))
		copy(out.Precommits, c.Precommits)
	}
	if c.Prevotes != nil {
		out.Prevotes = make([]string, len(c.Prevotes))
		copy(out.Prevotes, c.Prevotes)
	}
	return out
}

type rawNodeState struct {
	Result *rawResultState `json:"result"`
}

type rawResultState struct {
	RoundState *CometNodeState `json:"round_state"`
}

func EmptyCometNodeState() *CometNodeState {
	return &CometNodeState{}
}

func (r *CometNodeState) Copy() *CometNodeState {
	out := &CometNodeState{
		Height:            r.Height,
		Round:             r.Round,
		Step:              r.Step,
		HeightRoundStep:   r.HeightRoundStep,
		ProposalBlockHash: r.ProposalBlockHash,
		LockedBlockHash:   r.LockedBlockHash,
		ValidBlockHash:    r.ValidBlockHash,
		LogStdout:         r.LogStdout,
		LogStderr:         r.LogStderr,
	}
	if r.Votes != nil {
		out.Votes = make([]*CometNodeVotes, len(r.Votes))
		for i, v := range r.Votes {
			out.Votes[i] = v.Copy()
		}
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
	active  bool

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
		"--home", r.config.WorkingDir,
		"--proxy_app", "persistent_kvstore",
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
	r.active = true
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
	r.active = false
	r.ctx = nil
	r.cancel = func() {}
	r.process = nil

	return err
}

func (r *CometNode) Terminate() error {
	err := r.Stop()
	r.Cleanup()

	// stdout := strings.ToLower(r.stdout.String())
	// stderr := strings.ToLower(r.stderr.String())

	// Todo: check for bugs in the logs
	return err
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

	stateRaw := &rawNodeState{}
	err = json.Unmarshal(stateBS, &stateRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to read state from node: %s", err)
	}

	newState := stateRaw.Result.RoundState.Copy()
	hrs := strings.Split(newState.HeightRoundStep, "/")
	if len(hrs) == 3 {
		newState.Height, _ = strconv.Atoi(hrs[0])
		newState.Round, _ = strconv.Atoi(hrs[1])
		step, _ := strconv.Atoi(hrs[2])
		switch step {
		case 1:
			newState.Step = "RoundStepNewHeight"
		case 2:
			newState.Step = "RoundStepNewRound"
		case 3:
			newState.Step = "RoundStepPropose"
		case 4:
			newState.Step = "RoundStepPrevote"
		case 5:
			newState.Step = "RoundStepPrevoteWait"
		case 6:
			newState.Step = "RoundStepPrecommit"
		case 7:
			newState.Step = "RoundStepPrecommitWait"
		case 8:
			newState.Step = "RoundStepCommit"
		default:
			newState.Step = "RoundStepUnknown"
		}
	}

	newState.LogStdout = r.stdout.String()
	newState.LogStderr = r.stderr.String()

	return newState, nil
}

func (r *CometNode) IsActive() bool {
	return r.active
}

func (r *CometNode) SendRequest(req string) error {
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
	resp, err := client.Get("http://" + r.config.RPCAddress + "/" + req)
	if err != nil {
		return fmt.Errorf("error sending request: %s", err)
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)
	return nil
}

type CometClusterConfig struct {
	NumNodes            int
	CometBinaryPath     string
	InterceptListenPort int
	BaseRPCPort         int
	BaseWorkingDir      string
	NumRequests         int
	CreateEmptyBlocks   bool

	TimeoutPropose   int
	TimeoutPrevote   int
	TimeoutPrecommit int
	TimeoutCommit    int
}

func (c *CometClusterConfig) SetDefaults() {
	if c.TimeoutPropose == 0 {
		c.TimeoutPropose = 100
	}
	if c.TimeoutPrevote == 0 {
		c.TimeoutPrevote = 20
	}
	if c.TimeoutPrecommit == 0 {
		c.TimeoutPrecommit = 20
	}
	if c.TimeoutCommit == 0 {
		c.TimeoutCommit = 20
	}
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

func NewCluster(config *CometClusterConfig) (*CometCluster, error) {
	config.SetDefaults()
	c := &CometCluster{
		config: config,
		Nodes:  make(map[int]*CometNode),
	}
	// Make nodes
	for i := 1; i <= c.config.NumNodes; i++ {
		nConfig := config.GetNodeConfig(i)
		c.Nodes[i] = NewCometNode(nConfig)
	}

	if _, err := os.Stat(config.BaseWorkingDir); err != nil {
		os.MkdirAll(config.BaseWorkingDir, 0750)
	}

	testnetArgs := []string{
		"itestnet",
		"--o", c.config.BaseWorkingDir,
		"--v", strconv.Itoa(c.config.NumNodes),
		"--timeout-propose", strconv.Itoa(c.config.TimeoutPropose),
		"--timeout-prevote", strconv.Itoa(c.config.TimeoutPrevote),
		"--timeout-precommit", strconv.Itoa(c.config.TimeoutPrecommit),
		"--timeout-commit", strconv.Itoa(c.config.TimeoutCommit),
		"--debug",
	}
	if config.CreateEmptyBlocks {
		testnetArgs = append(testnetArgs, "--create-empty-blocks")
	}

	cmd := exec.Command(config.CometBinaryPath, testnetArgs...)
	stdout := new(bytes.Buffer)
	cmd.Stdout = stdout
	stderr := new(bytes.Buffer)
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		fmt.Println(stdout.String())
		fmt.Println(stderr.String())
		return nil, fmt.Errorf("error creating config: %s", err)
	}

	return c, nil
}

func (c *CometCluster) GetNodeStates() map[uint64]*CometNodeState {
	out := make(map[uint64]*CometNodeState)
	for id, node := range c.Nodes {
		state, err := node.Info()
		if err != nil {
			e := EmptyCometNodeState()
			e.LogStdout = node.stdout.String()
			e.LogStderr = node.stderr.String()

			out[uint64(id)] = e
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
		e := node.Terminate()
		if e != nil {
			err = e
		}
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

func (c *CometCluster) Execute(req string) error {
	var activeNode *CometNode = nil
	for i := 1; i <= c.config.NumNodes; i++ {
		node, ok := c.Nodes[i]
		if ok && node.IsActive() {
			activeNode = node
			break
		}
	}
	if activeNode == nil {
		return errors.New("no active node")
	}

	return activeNode.SendRequest(req)
}
