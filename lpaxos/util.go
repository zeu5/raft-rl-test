package lpaxos

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type Entry struct {
	Data []byte
}

func LogHash(entries []Entry) string {
	bs, _ := json.Marshal(entries)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

type Log struct {
	entries []Entry
}

func NewLog() *Log {
	return &Log{
		entries: make([]Entry, 0),
	}

}

func (l *Log) Add(e Entry) {
	l.entries = append(l.entries, e)
}

func (l *Log) Replace(new []Entry) {
	l.entries = new
}

func (l *Log) Entries() []Entry {
	return l.entries
}

func (l *Log) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.entries)
}

func (l *Log) Hash() string {
	return LogHash(l.entries)
}

type Ack struct {
	Peer    uint64
	Phase   int
	Last    int
	Log     []Entry
	LogHash string
}

type Promise struct {
	Peer    uint64
	Phase   int
	Log     []Entry
	LogHash string
}

type Tracker struct {
	Acks        map[uint64]Ack
	Promises    map[uint64]Promise
	PhaseChange map[uint64]Ack
}

func NewTracker() *Tracker {
	return &Tracker{
		Acks:     make(map[uint64]Ack),
		Promises: make(map[uint64]Promise),
	}
}

func (t *Tracker) TrackAck(a Ack) {
	cur, ok := t.Acks[a.Peer]
	if !ok {
		t.Acks[a.Peer] = a
		return
	}
	if a.Phase < cur.Phase {
		// Old, ignore
		return
	}
	if a.Phase == cur.Phase && (a.LogHash != cur.LogHash || a.Last != cur.Last) {
		// Duplicate mismatching ACK
		return
	}
	t.Acks[a.Peer] = a
}

func (t *Tracker) TrackPromise(p Promise) {
	cur, ok := t.Promises[p.Peer]
	if !ok {
		t.Promises[p.Peer] = p
		return
	}
	if p.Phase < cur.Phase {
		// Old, ignore
		return
	}
	if p.Phase == cur.Phase && p.LogHash != cur.LogHash {
		// Duplicate for different log
		// Conflict
		return
	}
	t.Promises[p.Peer] = p
}

func (t *Tracker) TrackPhaseChange(a Ack) {
	cur, ok := t.PhaseChange[a.Peer]
	if !ok {
		t.PhaseChange[a.Peer] = a
		return
	}
	if a.Phase < cur.Phase {
		return
	}
	t.PhaseChange[a.Peer] = a
}

func (t *Tracker) ValidAcks(phase int) int {
	count := 0
	for _, a := range t.Acks {
		if a.Phase == phase {
			count += 1
		}
	}
	return count
}

func (t *Tracker) ValidPromises(phase int, logHash string) int {
	count := 0
	for _, p := range t.Promises {
		if p.Phase == phase && p.LogHash == logHash {
			count += 1
		}
	}
	return count
}

func (t *Tracker) ValidPhaseChanges(phase int) int {
	count := 0
	for _, a := range t.PhaseChange {
		if a.Phase == phase {
			count += 1
		}
	}
	return count
}

func (t *Tracker) Reset(phase int) {
	newAcks := make(map[uint64]Ack)
	newPromises := make(map[uint64]Promise)
	newPhaseChange := make(map[uint64]Ack)

	for _, a := range t.Acks {
		if a.Phase >= phase {
			newAcks[a.Peer] = a
		}
	}
	for _, p := range t.Promises {
		if p.Phase >= phase {
			newPromises[p.Peer] = p
		}
	}
	for _, a := range t.PhaseChange {
		if a.Phase > phase {
			newPhaseChange[a.Peer] = a
		} else if a.Phase == phase {
			newAcks[a.Peer] = a
		}
	}
	t.Acks = newAcks
	t.Promises = newPromises
	t.PhaseChange = newPhaseChange
}
