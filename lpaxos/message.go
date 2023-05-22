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
	F       uint64
	T       uint64
	Last    int
	Log     []Entry
	LogHash string
}

func (m Message) Copy() Message {
	newM := Message{
		Type:    m.Type,
		Phase:   m.Phase,
		F:       m.F,
		T:       m.T,
		Last:    m.Last,
		Log:     make([]Entry, len(m.Log)),
		LogHash: m.LogHash,
	}
	for i, e := range m.Log {
		newM.Log[i] = e.Copy()
	}
	return newM
}

func (m Message) From() uint64 {
	return m.F
}

func (m Message) To() uint64 {
	return m.T
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
