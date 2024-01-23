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

func ColorHeight() CometColorFunc {
	return func(cns *CometNodeState) (string, interface{}) {
		return "height", cns.Height
	}
}

func ColorStep() CometColorFunc {
	return func(cns *CometNodeState) (string, interface{}) {
		return "step", cns.Step
	}
}

func ColorRound() CometColorFunc {
	return func(cns *CometNodeState) (string, interface{}) {
		return "round", cns.Round
	}
}

func ColorProposal() CometColorFunc {
	return func(s *CometNodeState) (string, interface{}) {
		return "have_proposal", s.ProposalBlockHash != ""
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
		votes := make([][]int, 0)
		for _, v := range s.Votes {
			prevotes := 0
			precommits := 0
			for _, pv := range v.Prevotes {
				if pv != "nil-Vote" {
					prevotes += 1
				}
			}
			for _, pcv := range v.Precommits {
				if pcv != "nil-Vote" {
					precommits += 1
				}
			}
			votes = append(votes, []int{prevotes, precommits})
		}
		return "num_votes", votes
	}
}

func ColorCurRoundVotes() CometColorFunc {
	return func(cns *CometNodeState) (string, interface{}) {
		if len(cns.Votes) <= cns.Round {
			return "round_votes", []int{}
		}
		votes := cns.Votes[cns.Round]
		prevotes := 0
		precommits := 0
		for _, pv := range votes.Prevotes {
			if pv != "nil-Vote" {
				prevotes += 1
			}
		}
		for _, pcv := range votes.Precommits {
			if pcv != "nil-Vote" {
				precommits += 1
			}
		}
		return "round_votes", []int{prevotes, precommits}
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
