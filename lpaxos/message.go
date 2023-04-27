package lpaxos

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type MessageType string

var (
	PrepareMessage MessageType = "Prepare"
	AckMessage     MessageType = "Ack"
	ProposeMessage MessageType = "Propose"
	PromiseMessage MessageType = "Promise"
	// Used only for forwarding commands to leader
	CommandMessage     MessageType = "Command"
	HeartbeatMessage   MessageType = "Heartbeat"
	PhaseChangeMessage MessageType = "PhaseChange"
)

type Message struct {
	Type    MessageType `json:"type"`
	Phase   int
	From    uint64
	To      uint64
	Last    int
	Log     []Entry
	LogHash string
}

func (m Message) Copy() Message {
	newM := Message{
		Type:    m.Type,
		Phase:   m.Phase,
		From:    m.From,
		To:      m.To,
		Last:    m.Last,
		Log:     make([]Entry, len(m.Log)),
		LogHash: m.LogHash,
	}
	for i, e := range m.Log {
		newM.Log[i] = e.Copy()
	}
	return newM
}

func (m Message) Hash() string {
	bs, _ := json.Marshal(m)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

func validMessage(m Message) bool {
	return true
	// if m.Log == nil || len(m.Log) == 0 {
	// 	return true
	// }
	// bs, _ := json.Marshal(m.Log)
	// hash := sha256.Sum256(bs)
	// return m.LogHash == hex.EncodeToString(hash[:])
}
