package lpaxos

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"

	"github.com/zeu5/raft-rl-test/types"
)

type LPaxosState struct {
	NodeStates   map[uint64]LNodeState
	Messages     map[string]Message
	WithTimeouts bool
}

var _ types.State = &LPaxosState{}

type LPaxosAction struct {
	Type    string
	Message Message `json:",omitempty"`
	Node    uint64  `json:",omitempty"`
}

var _ types.Action = &LPaxosAction{}

func (l *LPaxosAction) String() string {
	bs, _ := json.Marshal(l)
	return string(bs)
}

func (l *LPaxosAction) Hash() string {
	bs, _ := json.Marshal(l)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

func (l *LPaxosState) Actions() []types.Action {
	result := make([]types.Action, len(l.Messages))
	i := 0
	for _, m := range l.Messages {
		result[i] = &LPaxosAction{
			Type:    "Deliver",
			Message: m,
		}
	}
	if l.WithTimeouts {
		nodes := make(map[uint64]bool)
		for _, m := range l.Messages {
			if _, ok := nodes[m.To]; !ok {
				nodes[m.To] = true
			}
		}
		for n := range nodes {
			result = append(result, &LPaxosAction{
				Type: "Drop",
				Node: n,
			})
		}
	}
	return result
}

func (l *LPaxosState) Hash() string {
	bs, _ := json.Marshal(l)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

func (l *LPaxosState) NodeStateHash() string {
	bs, _ := json.Marshal(l.NodeStates)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

type LPaxosEnvConfig struct {
	Replicas int
	Timeouts bool
	Timeout  int
	Requests int
}

type LPaxosEnv struct {
	config   LPaxosEnvConfig
	nodes    map[uint64]*LPaxosNode
	messages map[string]Message
	curState *LPaxosState
}

var _ types.Environment = &LPaxosEnv{}

func NewLPaxosEnv(c LPaxosEnvConfig) *LPaxosEnv {
	e := &LPaxosEnv{
		config:   c,
		nodes:    make(map[uint64]*LPaxosNode),
		messages: make(map[string]Message),
	}
	for i := 0; i < c.Requests+1; i++ {
		cmd := Message{
			Type: CommandMessage,
			Log:  []Entry{{Data: []byte(strconv.Itoa(i + 1))}},
		}
		e.messages[cmd.Hash()] = cmd
	}
	e.makeNodes()
	return e
}

func (e *LPaxosEnv) makeNodes() {
	peers := make([]Peer, e.config.Replicas)
	for i := 0; i < e.config.Replicas; i++ {
		peers[i] = Peer{ID: uint64(i + 1)}
	}
	for i := 0; i < e.config.Replicas; i++ {
		id := uint64(i + 1)
		e.nodes[id] = NewLPaxosNode(LPaxosConfig{ID: id, Peers: peers, Timeout: e.config.Timeout})
	}
	initState := &LPaxosState{
		NodeStates:   make(map[uint64]LNodeState),
		Messages:     e.messages,
		WithTimeouts: e.config.Timeouts,
	}
	for id, n := range e.nodes {
		initState.NodeStates[id] = n.Status()
	}
	e.curState = initState
}

func (e *LPaxosEnv) Reset() types.State {
	e.messages = make(map[string]Message)
	cmd := Message{
		Type: CommandMessage,
		Log:  []Entry{{Data: []byte("1")}},
	}
	e.messages[cmd.Hash()] = cmd
	e.makeNodes()
	return e.curState
}

func (e *LPaxosEnv) Step(a types.Action) types.State {
	lAction := a.(*LPaxosAction)
	switch lAction.Type {
	case "Deliver":
		if lAction.Message.Type == CommandMessage {
			haveLeader := false
			var leader uint64 = 0
			for id, n := range e.curState.NodeStates {
				if n.Leader == id {
					leader = id
					haveLeader = true
					break
				}
			}
			if haveLeader {
				message := lAction.Message
				message.To = leader
				e.nodes[leader].Step(message)
				delete(e.messages, message.Hash())
			}
		} else {
			message := lAction.Message
			e.nodes[message.To].Step(message)
			delete(e.messages, message.Hash())
		}
		for _, node := range e.nodes {
			ticks := 2
			for i := 0; i < ticks; i++ {
				node.Tick()
			}
		}
		newState := &LPaxosState{
			NodeStates:   make(map[uint64]LNodeState),
			WithTimeouts: e.config.Timeouts,
		}
		for id, node := range e.nodes {
			rd := node.Ready()
			for _, m := range rd.Messages {
				e.messages[m.Hash()] = m
			}
			newState.NodeStates[id] = node.Status()
		}
		newState.Messages = e.messages
		return newState
	case "Drop":
		newMessages := make(map[string]Message)
		for key, message := range e.messages {
			if message.To != lAction.Node {
				newMessages[key] = message
			}
		}
		newState := &LPaxosState{
			NodeStates:   e.curState.NodeStates,
			Messages:     newMessages,
			WithTimeouts: e.config.Timeouts,
		}
		e.curState = newState
		return newState
	}
	return nil
}
