package redisraft

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/zeu5/raft-rl-test/types"
)

type RedisPartitionColor struct {
	Params map[string]interface{}
}

var _ types.Color = &RedisPartitionColor{}

func (r *RedisPartitionColor) Hash() string {
	bs, _ := json.Marshal(r.Params)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

func (r *RedisPartitionColor) Copy() types.Color {
	new := &RedisPartitionColor{
		Params: make(map[string]interface{}),
	}
	for k, v := range r.Params {
		new.Params[k] = v
	}
	return new
}

// various functions to take info from RedisNodeState to 'colors' of the abstracted state
type RedisRaftColorFunc func(*RedisNodeState) (string, interface{})

// return the state of the node, should be one of these:
//
//	var stmap = [...]string{
//		"StateFollower",
//		"StateCandidate",
//		"StateLeader",
//		"StatePreCandidate",
//	}
func ColorState() RedisRaftColorFunc {
	return func(s *RedisNodeState) (string, interface{}) {
		return "state", s.State
	}
}

// return the current term of the node, should be just a number
func ColorTerm() RedisRaftColorFunc {
	return func(s *RedisNodeState) (string, interface{}) {
		return "term", s.Term
	}
}

// return the current term of the node, provides a bound that is the maximum considered term
func ColorBoundedTerm(bound int) RedisRaftColorFunc {
	return func(s *RedisNodeState) (string, interface{}) {
		term := s.Term
		if term > bound {
			term = bound
		}
		return "boundedTerm", term
	}
}

// return the relative current term of the node, 1 is the current minimum term number, the others are computed as difference from this
func ColorRelativeTerm(minimum int) RedisRaftColorFunc {
	return func(s *RedisNodeState) (string, interface{}) {
		term := s.Term            // get the value from process Status
		term = term - minimum + 1 // compute difference
		return "boundedTerm", term
	}
}

// return the relative current term of the node, 1 is the current minimum term number, the others are computed as difference from this
// bound provides the maximum considered term gap
func ColorRelativeBoundedTerm(minimum int, bound int) RedisRaftColorFunc {
	return func(s *RedisNodeState) (string, interface{}) {
		term := s.Term            // get the value from process Status
		term = term - minimum + 1 // compute difference
		if term > bound {         // set to bound if higher gap
			term = bound
		}
		return "boundedTerm", term
	}
}

func ColorCommit() RedisRaftColorFunc {
	return func(s *RedisNodeState) (string, interface{}) {
		return "commit", s.Commit
	}
}

func ColorApplied() RedisRaftColorFunc {
	return func(s *RedisNodeState) (string, interface{}) {
		return "applied", s.Applied
	}
}

func ColorVote() RedisRaftColorFunc {
	return func(s *RedisNodeState) (string, interface{}) {
		return "vote", s.Vote
	}
}

func ColorLeader() RedisRaftColorFunc {
	return func(s *RedisNodeState) (string, interface{}) {
		return "leader", s.Lead
	}
}

func ColorIndex() RedisRaftColorFunc {
	return func(s *RedisNodeState) (string, interface{}) {
		return "index", s.Index
	}
}

func ColorSnapshot() RedisRaftColorFunc {
	return func(rns *RedisNodeState) (string, interface{}) {
		return "snapshot", rns.Snapshot
	}
}

// Abstraction for the log, it consists of the list of entries considering only their Term and Type
func ColorLog() RedisRaftColorFunc {
	return func(rns *RedisNodeState) (string, interface{}) {
		result := make([]string, 0)
		for _, entry := range rns.Logs {
			entryVal := fmt.Sprintf("%d-%d", entry.Term, entry.Type) // the log is seen as a list of term-type elements
			result = append(result, entryVal)
		}
		return "log", result
	}
}

// Abstraction for the log, it consists of the list of entries considering only their Term (bounded to a given limit) and Type
func ColorBoundedLog(termLimit int) RedisRaftColorFunc {
	return func(rns *RedisNodeState) (string, interface{}) {
		result := make([]string, 0)
		for _, entry := range rns.Logs {
			term := entry.Term
			if entry.Term > termLimit {
				term = termLimit + 1
			}
			entryVal := fmt.Sprintf("%d-%d", term, entry.Type) // the log is seen as a list of term-type elements
			result = append(result, entryVal)
		}
		return "log", result
	}
}

type RedisRaftStatePainter struct {
	paramFuncs []RedisRaftColorFunc
}

func NewRedisRaftStatePainter(paramFuncs ...RedisRaftColorFunc) *RedisRaftStatePainter {
	return &RedisRaftStatePainter{
		paramFuncs: paramFuncs,
	}
}

// apply abstraction on a ReplicaState
func (p *RedisRaftStatePainter) Color(s types.ReplicaState) types.Color {
	rs := s.(*RedisNodeState) // cast the "state" component into RedisNodeState
	c := &RedisPartitionColor{
		Params: make(map[string]interface{}),
	}

	for _, p := range p.paramFuncs {
		k, v := p(rs)
		c.Params[k] = v
	}
	return c
}

var _ types.Painter = &RedisRaftStatePainter{}
