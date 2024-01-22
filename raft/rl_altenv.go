package raft

import (
	"github.com/zeu5/raft-rl-test/types"
	raft "go.etcd.io/raft/v3"
	pb "go.etcd.io/raft/v3/raftpb"
)

type AbsRaftEnvironment struct {
	*RaftEnvironment
	abstracter StateAbstracter
}

var _ types.Environment = (*AbsRaftEnvironment)(nil)

func NewAbsRaftEnvironment(config RaftEnvironmentConfig, abstracter StateAbstracter) *AbsRaftEnvironment {
	r := &AbsRaftEnvironment{
		RaftEnvironment: NewRaftEnvironment(config),
		abstracter:      abstracter,
	}
	return r
}

type StateAbstracter func(raft.Status) raft.Status

func DefaultAbstractor() StateAbstracter {
	return func(s raft.Status) raft.Status {
		return s
	}
}

func IgnoreTerm() StateAbstracter {
	return func(s raft.Status) raft.Status {
		s.Term = 0
		return s
	}
}

func IgnoreTermUnlessLeader() StateAbstracter {
	return func(s raft.Status) raft.Status {
		if s.RaftState != raft.StateLeader {
			s.Term = 0
		}
		return s
	}
}

func IgnoreVote() StateAbstracter {
	return func(s raft.Status) raft.Status {
		s.Vote = 0
		return s
	}
}

func (r *AbsRaftEnvironment) Step(action types.Action, _ *types.EpisodeContext) (types.State, error) {
	raftAction := action.(*RaftAction)
	switch raftAction.Type {
	case "DeliverMessage":
		message := raftAction.Message
		if message.Type == pb.MsgProp {
			haveLeader := false
			leader := uint64(0)
			for id, node := range r.nodes {
				if node.Status().RaftState == raft.StateLeader {
					haveLeader = true
					leader = id
					break
				}
			}
			if haveLeader {
				message.To = leader
				r.nodes[leader].Step((message))
				delete(r.messages, msgKey(message))
			}
		} else {
			node := r.nodes[message.To]
			node.Step(message)
			delete(r.messages, msgKey(message))
		}
		// Take random number of ticks and update node states
		for _, node := range r.nodes {
			node.Tick()
		}
		newState := RaftState{
			NodeStates:   make(map[uint64]raft.Status),
			WithTimeouts: r.config.Timeouts,
		}
		for id, node := range r.nodes {
			if node.HasReady() {
				ready := node.Ready()
				if !raft.IsEmptySnap(ready.Snapshot) {
					r.storages[id].ApplySnapshot(ready.Snapshot)
				}
				r.storages[id].Append(ready.Entries)
				for _, message := range ready.Messages {
					r.messages[msgKey(message)] = message
				}
				node.Advance(ready)
			}
			newState.NodeStates[id] = r.abstracter(node.Status())
		}
		newState.Messages = copyMessages(r.messages)
		r.curState = newState
		return newState, nil
	case "TimeoutProcess":
		newMessages := make(map[string]pb.Message)
		for key, message := range r.messages {
			if message.To != raftAction.Replica {
				newMessages[key] = message
			}
		}
		newState := RaftState{
			NodeStates:   make(map[uint64]raft.Status),
			Messages:     copyMessages(newMessages),
			WithTimeouts: r.config.Timeouts,
		}

		for id, node := range r.nodes {
			newState.NodeStates[id] = r.abstracter(node.Status())
		}
		r.curState = newState
		return newState, nil
	}
	return nil, nil
}
