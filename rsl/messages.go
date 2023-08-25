package rsl

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type MessageType string

var (
	MessageAcceptReq   MessageType = "AcceptReq"
	MessageAcceptResp  MessageType = "AcceptResp"
	MessageStatusReq   MessageType = "StatusReq"
	MessageStatusResp  MessageType = "StatusResp"
	MessagePrepareReq  MessageType = "PrepareReq"
	MessagePrepareResp MessageType = "PrepareResp"
	MessageFetchReq    MessageType = "FetchReq"
	MessageFetchResp   MessageType = "FetchResp"
	MessageCommand     MessageType = "CommandMessage"
)

type Message struct {
	From           uint64
	To             uint64
	Type           MessageType
	Ballot         Ballot
	Decree         int
	Command        Command
	ProposalBallot Ballot
	Timestamp      int
	Accept         bool
	Proposals      []Proposal
}

func (m Message) Hash() string {
	bs, _ := json.Marshal(m)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

func (m Message) Copy() Message {
	return Message{
		From:   m.From,
		To:     m.To,
		Type:   m.Type,
		Ballot: m.Ballot.Copy(),
		Decree: m.Decree,
		Command: Command{
			Data: bytes.Clone(m.Command.Data),
		},
	}
}

func (n *Node) broadcast(m Message) {
	m.From = n.ID
	for _, p := range n.config.Peers {
		if p == n.ID {
			continue
		}
		mC := m.Copy()
		mC.To = p
		n.pendingMessagesToSend = append(n.pendingMessagesToSend, mC)
	}
}

func (n *Node) send(m Message) {
	m.From = n.ID
	n.pendingMessagesToSend = append(n.pendingMessagesToSend, m.Copy())
}
