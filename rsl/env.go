package rsl

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"

	"github.com/zeu5/raft-rl-test/types"
)

type MessageWrapper struct {
	m Message
}

func (m MessageWrapper) From() uint64 {
	return m.m.From
}

func (m MessageWrapper) To() uint64 {
	return m.m.To
}

func (m MessageWrapper) Hash() string {
	return m.m.Hash()
}

var _ types.Message = MessageWrapper{}

type RSLPartitionState struct {
	ReplicaStates map[uint64]LocalState
	Messages      map[string]Message
	Requests      []Message
}

func (p *RSLPartitionState) GetReplicaState(node uint64) types.ReplicaState {
	s, ok := p.ReplicaStates[node]
	if !ok {
		return EmptyLocalState()
	}
	return s.Copy()
}

func (p *RSLPartitionState) PendingMessages() map[string]types.Message {
	return copyToMessageWrappers(p.Messages)
}

func (p *RSLPartitionState) CanDeliverRequest() bool {
	havePrimary := false
	for _, s := range p.ReplicaStates {
		if s.State == StateStablePrimary {
			havePrimary = true
			break
		}
	}
	return havePrimary
}

func (p *RSLPartitionState) PendingRequests() []types.Request {
	out := make([]types.Request, len(p.Requests))
	for i, r := range p.Requests {
		out[i] = r.Copy()
	}
	return out
}

func copyToMessageWrappers(messages map[string]Message) map[string]types.Message {
	out := make(map[string]types.Message)
	for hash, m := range messages {
		out[hash] = MessageWrapper{
			m: m.Copy(),
		}
	}
	return out
}

var _ types.PartitionedSystemState = &RSLPartitionState{}

type RSLEnvConfig struct {
	Nodes              int
	NodeConfig         NodeConfig
	NumCommands        int
	AdditionalCommands []Command
}

// RSL partition environment
// structure that wraps the nodes to fir the RL environment
type RSLPartitionEnv struct {
	config   RSLEnvConfig
	nodes    map[uint64]*Node
	messages map[string]Message
	curState *RSLPartitionState
}

var _ types.PartitionedSystemEnvironment = &RSLPartitionEnv{}

func NewRLSPartitionEnv(c RSLEnvConfig) *RSLPartitionEnv {
	e := &RSLPartitionEnv{
		config:   c,
		nodes:    make(map[uint64]*Node),
		messages: make(map[string]Message),
	}
	e.Reset()
	return e
}

// Reset creates new nodes and clears all messages
func (r *RSLPartitionEnv) Reset() types.PartitionedSystemState {
	peers := make([]uint64, r.config.Nodes)
	for i := 0; i < r.config.Nodes; i++ {
		peers[i] = uint64(i + 1)
	}

	newState := &RSLPartitionState{
		ReplicaStates: make(map[uint64]LocalState),
		Messages:      copyMessages(r.messages),
		Requests:      make([]Message, r.config.NumCommands),
	}

	r.config.NodeConfig.Peers = peers
	for i := 0; i < r.config.Nodes; i++ {
		cfg := r.config.NodeConfig.Copy()
		cfg.ID = uint64(i + 1)
		node := NewNode(&cfg)
		r.nodes[node.ID] = node
		newState.ReplicaStates[node.ID] = node.State()

		node.Start()
	}
	for i := 0; i < r.config.NumCommands; i++ {
		cmd := Message{Type: MessageCommand, Command: Command{Data: []byte(strconv.Itoa(i + 1))}}
		newState.Requests[i] = cmd
	}
	for _, c := range r.config.AdditionalCommands {
		cmd := Message{Type: MessageCommand, Command: c.Copy()}
		newState.Requests = append(newState.Requests, cmd)
	}
	r.curState = newState
	return newState
}

func (r *RSLPartitionEnv) Start(nodeID uint64) {
	_, ok := r.nodes[nodeID]
	if ok {
		// Node already started
		return
	}
	cfg := r.config.NodeConfig.Copy()
	cfg.ID = nodeID
	node := NewNode(&cfg)
	r.nodes[nodeID] = node

	node.Start()
}

func (r *RSLPartitionEnv) Stop(node uint64) {
	delete(r.nodes, node)
}

func (r *RSLPartitionEnv) ReceiveRequest(req types.Request) types.PartitionedSystemState {
	newState := &RSLPartitionState{
		ReplicaStates: copyReplicaStates(r.curState.ReplicaStates),
		Messages:      copyMessages(r.messages),
		Requests:      make([]Message, 0),
	}
	remainingRequests := r.curState.Requests
	cmd := req.(Message)
	for _, n := range r.nodes {
		if n.IsPrimary() {
			n.Propose(cmd.Command)
			remainingRequests = r.curState.Requests[1:]
			break
		}
	}
	for _, re := range remainingRequests {
		newState.Requests = append(newState.Requests, re.Copy())
	}
	r.curState = newState
	return newState
}

// Moves clock of each process by one
// Checks for messages after clock has advanced
func (r *RSLPartitionEnv) Tick() types.PartitionedSystemState {
	newState := &RSLPartitionState{
		ReplicaStates: make(map[uint64]LocalState),
		Messages:      make(map[string]Message),
	}
	for _, node := range r.nodes {
		node.Tick()
		if node.HasReady() {
			rd := node.Ready()
			for _, m := range rd.Messages {
				r.messages[m.Hash()] = m
			}
		}
		newState.ReplicaStates[node.ID] = node.State()
	}
	newState.Messages = copyMessages(r.messages)
	newState.Requests = copyMessagesList(r.curState.Requests)
	r.curState = newState
	return newState
}

func (l *RSLPartitionEnv) DeliverMessages(messages []types.Message) types.PartitionedSystemState {
	var s types.PartitionedSystemState = nil
	for _, m := range messages {
		s = l.deliverMessage(m)
	}
	return s
}

// Deliver the message
func (r *RSLPartitionEnv) deliverMessage(m types.Message) types.PartitionedSystemState {
	// Node that m is of type MessageWrapper
	message := m.(MessageWrapper).m

	if message.Type == MessageCommand {
		for _, n := range r.nodes {
			if n.IsPrimary() {
				n.Propose(message.Command)
				delete(r.messages, m.Hash())
			}
		}
	} else {
		node, ok := r.nodes[message.To]
		if ok {
			node.Step(message)
			delete(r.messages, message.Hash())
		}
	}

	newState := &RSLPartitionState{
		ReplicaStates: make(map[uint64]LocalState),
		Messages:      make(map[string]Message),
	}
	for id, node := range r.nodes {
		newState.ReplicaStates[id] = node.State()
		if node.HasReady() {
			rd := node.Ready()
			for _, m := range rd.Messages {
				r.messages[m.Hash()] = m
			}
		}
	}
	newState.Messages = copyMessages(r.messages)
	newState.Requests = copyMessagesList(r.curState.Requests)
	r.curState = newState

	return newState
}

func (l *RSLPartitionEnv) DropMessages(messages []types.Message) types.PartitionedSystemState {
	var s types.PartitionedSystemState
	for _, m := range messages {
		s = l.dropMessage(m)
	}
	return s
}

// Remove the message from the pool
func (r *RSLPartitionEnv) dropMessage(m types.Message) types.PartitionedSystemState {
	// Node that m is of type MessageWrapper
	message := m.(MessageWrapper).m
	delete(r.messages, message.Hash())
	newState := &RSLPartitionState{
		ReplicaStates: copyReplicaStates(r.curState.ReplicaStates),
		Messages:      copyMessages(r.messages),
		Requests:      copyMessagesList(r.curState.Requests),
	}
	r.curState = newState
	return newState
}

type RSLColorFunc func(LocalState) (string, interface{})

// The color of each node/replica marked by a key value
// The Hash function returns the color hash
type RSLColor struct {
	Params map[string]interface{}
}

func (r *RSLColor) Hash() string {
	bs, _ := json.Marshal(r)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

func (r *RSLColor) Copy() types.Color {
	out := &RSLColor{
		Params: make(map[string]interface{}),
	}
	for k, val := range r.Params {
		out.Params[k] = val
	}
	return out
}

// Painter to paint nodes with colors
// The state abstraction is useful here
type RSLPainter struct {
	colorFuncs []RSLColorFunc
}

func (r *RSLPainter) Color(s types.ReplicaState) types.Color {
	rslState := s.(LocalState)
	c := &RSLColor{
		Params: make(map[string]interface{}),
	}
	for _, f := range r.colorFuncs {
		key, val := f(rslState)
		c.Params[key] = val
	}
	return c
}

var _ types.Painter = &RSLPainter{}

func NewRSLPainter(colors ...RSLColorFunc) *RSLPainter {
	return &RSLPainter{
		colorFuncs: colors,
	}
}

func ColorState() RSLColorFunc {
	return func(ls LocalState) (string, interface{}) {
		return "state", string(ls.State)
	}
}

func ColorBoundedBallot(bound int) RSLColorFunc {
	return func(ls LocalState) (string, interface{}) {
		b := ls.MaxAcceptedProposal.Ballot.Num
		if b > bound {
			b = bound
		}
		return "boundedBallot", b
	}
}

func ColorBallot() RSLColorFunc {
	return func(ls LocalState) (string, interface{}) {
		return "ballot", ls.MaxAcceptedProposal.Ballot.Num
	}
}

func ColorDecree() RSLColorFunc {
	return func(ls LocalState) (string, interface{}) {
		return "decree", ls.MaxAcceptedProposal.Decree
	}
}

func ColorDecided() RSLColorFunc {
	return func(ls LocalState) (string, interface{}) {
		return "decided", ls.Decided
	}
}

func ColorLogLength() RSLColorFunc {
	return func(ls LocalState) (string, interface{}) {
		return "logLength", ls.Log.NumDecided()
	}
}
