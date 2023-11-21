package cbft

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/zeu5/raft-rl-test/types"
)

type CometPartitionColor struct {
	Params map[string]interface{}
}

var _ types.Color = &CometPartitionColor{}

func (r *CometPartitionColor) Hash() string {
	bs, _ := json.Marshal(r.Params)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

func (r *CometPartitionColor) Copy() types.Color {
	new := &CometPartitionColor{
		Params: make(map[string]interface{}),
	}
	for k, v := range r.Params {
		new.Params[k] = v
	}
	return new
}

// various functions to take info from CometNodeState to 'colors' of the abstracted state
type CometColorFunc func(*CometNodeState) (string, interface{})

// return the state of the node, should be one of these:
//
//	var stmap = [...]string{
//		"StateFollower",
//		"StateCandidate",
//		"StateLeader",
//		"StatePreCandidate",
//	}
func ColorHRS() CometColorFunc {
	return func(s *CometNodeState) (string, interface{}) {
		return "hrs", s.HeightRoundStep
	}
}

func ColorProposal() CometColorFunc {
	return func(s *CometNodeState) (string, interface{}) {
		return "proposal", s.ProposalBlockHash
	}
}

func ColorLockedValue() CometColorFunc {
	return func(s *CometNodeState) (string, interface{}) {
		return "locked", s.LockedBlockHash
	}
}

func ColorValidValue() CometColorFunc {
	return func(s *CometNodeState) (string, interface{}) {
		return "valid", s.ValidBlockHash
	}
}

func ColorNumVotes() CometColorFunc {
	return func(s *CometNodeState) (string, interface{}) {
		return "num_votes", len(s.Votes)
	}
}

func ColorVotes() CometColorFunc {
	return func(s *CometNodeState) (string, interface{}) {
		bs, _ := json.Marshal(s.Votes)
		hash := sha256.Sum256(bs)
		return "votes", hex.EncodeToString(hash[:])
	}
}

func ColorProposer() CometColorFunc {
	return func(s *CometNodeState) (string, interface{}) {
		proposerIndex := 0
		if s.Proposer != nil {
			proposerIndex = s.Proposer.Index
		}
		return "proposer", proposerIndex
	}
}

type CometStatePainter struct {
	paramFuncs []CometColorFunc
}

func NewCometStatePainter(paramFuncs ...CometColorFunc) *CometStatePainter {
	return &CometStatePainter{
		paramFuncs: paramFuncs,
	}
}

// apply abstraction on a ReplicaState
func (p *CometStatePainter) Color(s types.ReplicaState) types.Color {
	rs := s.(*CometNodeState) // cast the "state" component into CometNodeState
	c := &CometPartitionColor{
		Params: make(map[string]interface{}),
	}

	for _, p := range p.paramFuncs {
		k, v := p(rs)
		c.Params[k] = v
	}
	return c
}

var _ types.Painter = &CometStatePainter{}

func copyRequests(in []CometRequest) []CometRequest {
	out := make([]CometRequest, len(in))
	for i, r := range in {
		out[i] = r.Copy()
	}
	return out
}
