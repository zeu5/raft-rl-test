package lpaxos

import "github.com/zeu5/raft-rl-test/types"

type LPaxosAbsEnv struct {
	*LPaxosEnv
	abstracter Abstracter
}

type Abstracter func(LNodeState) LNodeState

func NewLPaxosAbsEnv(config LPaxosEnvConfig, abs Abstracter) *LPaxosAbsEnv {
	return &LPaxosAbsEnv{
		LPaxosEnv:  NewLPaxosEnv(config),
		abstracter: abs,
	}
}

func (e *LPaxosAbsEnv) Step(a types.Action) types.State {
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
		return newState
	case "Drop":
		newMessages := make(map[string]Message)
		for key, message := range e.messages {
			if message.To != lAction.Node {
				newMessages[key] = message
			}
		}
		newState := &LPaxosState{
			NodeStates:   copyNodeStates(e.curState.NodeStates),
			Messages:     copyMessages(newMessages),
			WithTimeouts: e.config.Timeouts,
		}
		e.curState = newState
		return newState
	}
	return nil
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
