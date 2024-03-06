package lpaxos

import "github.com/zeu5/raft-rl-test/types"

type LPaxosAbsEnv struct {
	*LPaxosEnv
	abstracter Abstracter
}

// var _ types.Environment = (*LPaxosAbsEnv)(nil)

type Abstracter func(LNodeState) LNodeState

func NewLPaxosAbsEnv(config LPaxosEnvConfig, abs Abstracter) *LPaxosAbsEnv {
	return &LPaxosAbsEnv{
		LPaxosEnv:  NewLPaxosEnv(config),
		abstracter: abs,
	}
}

func (e *LPaxosAbsEnv) Step(a types.Action, _ *types.StepContext) (types.State, error) {
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
				message.T = leader
				e.nodes[leader].Step(message)
				delete(e.messages, message.Hash())
			}
		} else {
			message := lAction.Message
			e.nodes[message.T].Step(message)
			delete(e.messages, message.Hash())
		}
		for _, node := range e.nodes {
			ticks := 1
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
			newState.NodeStates[id] = e.abstracter(node.Status())
		}
		newState.Messages = copyMessages(e.messages)
		e.curState = newState
		return newState, nil
	case "Drop":
		newMessages := make(map[string]Message)
		for key, message := range e.messages {
			if message.T != lAction.Node {
				newMessages[key] = message
			}
		}
		newState := &LPaxosState{
			NodeStates:   copyNodeStates(e.curState.NodeStates),
			Messages:     copyMessages(newMessages),
			WithTimeouts: e.config.Timeouts,
		}
		e.curState = newState
		return newState, nil
	}
	return nil, nil
}

func DefaultAbstractor() Abstracter {
	return func(ls LNodeState) LNodeState {
		return ls
	}
}

func IgnorePhase() Abstracter {
	return func(ls LNodeState) LNodeState {
		ls.Phase = 0
		return ls
	}
}

func IgnoreLast() Abstracter {
	return func(ls LNodeState) LNodeState {
		ls.Last = -1
		return ls
	}
}
