package cbft

import (
	"math"

	"github.com/zeu5/raft-rl-test/types"
)

func AnyReachedRound(round int) types.RewardFuncSingle {
	return func(s types.State) bool {
		if s == nil {
			return false
		}
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if ns.Round >= round {
				return true
			}
		}
		return false
	}
}

func MajorityNoProposal() types.RewardFuncSingle {
	return func(s types.State) bool {
		if s == nil {
			return false
		}
		total := 0
		haveProposal := 0
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if ns.ProposalBlockHash != "" || ns.Proposer != nil {
				haveProposal += 1
			}
			total += 1
		}
		return (total - haveProposal) >= (2*total/3 + 1)
	}
}

func AnyReachedStep(step string) types.RewardFuncSingle {
	return func(s types.State) bool {
		if s == nil {
			return false
		}
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if ns.Step == step {
				return true
			}
		}
		return false
	}
}

func AllInRound(round int) types.RewardFuncSingle {
	return func(s types.State) bool {
		if s == nil {
			return false
		}
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if ns.Round != round {
				return false
			}
		}
		return true
	}
}

func AllAtLeastRound(round int) types.RewardFuncSingle {
	return func(s types.State) bool {
		if s == nil {
			return false
		}
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if ns.Round < round {
				return false
			}
		}
		return true
	}
}

func MinRoundDifference(diff int) types.RewardFuncSingle {
	return func(s types.State) bool {
		if s == nil {
			return false
		}
		maxRound := -1
		minRound := math.MaxInt
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if ns.Round > maxRound {
				maxRound = ns.Round
			}
			if ns.Round < minRound {
				minRound = ns.Round
			}
		}
		return (maxRound - minRound) >= diff
	}
}

func AtLeastHeight(height int) types.RewardFuncSingle {
	return func(s types.State) bool {
		if s == nil {
			return false
		}
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if ns.Height < height {
				return false
			}
		}
		return true
	}
}

func AnyAtHeight(height int) types.RewardFuncSingle {
	return func(s types.State) bool {
		if s == nil {
			return false
		}
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if ns.Height == height {
				return true
			}
		}
		return false
	}
}

func AnyAtLeastHeight(height int) types.RewardFuncSingle {
	return func(s types.State) bool {
		if s == nil {
			return false
		}
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if ns.Height >= height {
				return true
			}
		}
		return false
	}
}

type NodeProperty func(*CometNodeState) bool

func (n NodeProperty) And(other NodeProperty) NodeProperty {
	return func(cns *CometNodeState) bool {
		return n(cns) && other(cns)
	}
}

func (n NodeProperty) Or(other NodeProperty) NodeProperty {
	return func(cns *CometNodeState) bool {
		return n(cns) || other(cns)
	}
}

func (n NodeProperty) Not() NodeProperty {
	return func(cns *CometNodeState) bool {
		return !n(cns)
	}
}

func NodeHasProperty(nodeProperty NodeProperty) types.RewardFuncSingle {
	return func(s types.State) bool {
		if s == nil {
			return false
		}
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if nodeProperty(ns) {
				return true
			}
		}
		return false
	}
}

func NodeInRound(round int) NodeProperty {
	return func(cns *CometNodeState) bool {
		return cns.Round == round
	}
}

func NodeAtLeastRound(round int) NodeProperty {
	return func(cns *CometNodeState) bool {
		return cns.Round >= round
	}
}

func NodeInHeight(height int) NodeProperty {
	return func(cns *CometNodeState) bool {
		return cns.Height == height
	}
}

func NodeAtMostHeight(height int) NodeProperty {
	return func(cns *CometNodeState) bool {
		return cns.Height <= height
	}
}

func NodeHasQuorum(prevotesFlag bool) NodeProperty {
	return func(cns *CometNodeState) bool {
		votes := cns.Votes[cns.Round].Precommits
		if prevotesFlag {
			votes = cns.Votes[cns.Round].Prevotes
		}

		maj := (2 * len(votes) / 3) + 1
		voteMap := make(map[string]int)
		for _, v := range votes {
			// TODO: need to figure out how to check if we have a quorum of votes or not
			if _, ok := voteMap[v]; !ok {
				voteMap[v] = 0
			}
			voteMap[v] += 1
		}
		noVotes := 0
		if nv, ok := voteMap["nil-Vote"]; ok {
			noVotes = nv
		}
		if noVotes > (len(votes) / 3) {
			return false
		}
		for _, count := range voteMap {
			if count >= maj {
				return true
			}
		}
		return false
	}
}

func NodeAtLeastHeight(height int) NodeProperty {
	return func(cns *CometNodeState) bool {
		return cns.Height >= height
	}
}

func NodeHasLocked() NodeProperty {
	return func(cns *CometNodeState) bool {
		return cns.LockedBlockHash != ""
	}
}

func AllAtLeastHeight(height int) types.RewardFuncSingle {
	return func(s types.State) bool {
		if s == nil {
			return false
		}
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if ns.Height < height {
				return false
			}
		}
		return true
	}
}

func EmptyLockedForAll() types.RewardFuncSingle {
	return func(s types.State) bool {
		if s == nil {
			return false
		}
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if ns.LockedBlockHash != "" {
				return false
			}
		}
		return true
	}
}

func LockedForAll() types.RewardFuncSingle {
	return func(s types.State) bool {
		if s == nil {
			return false
		}
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if ns.LockedBlockHash == "" {
				return false
			}
		}
		return true
	}
}

func SameLockedForAll() types.RewardFuncSingle {
	return func(s types.State) bool {
		if s == nil {
			return false
		}
		lockedValues := make(map[string]bool)
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if ns.LockedBlockHash != "" {
				lockedValues[ns.LockedBlockHash] = true
			} else {
				return false
			}
		}
		return len(lockedValues) == 1
	}
}

func DifferentLocked() types.RewardFuncSingle {
	return func(s types.State) bool {
		if s == nil {
			return false
		}
		lockedValues := make(map[string]bool)
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			if ns.LockedBlockHash != "" {
				lockedValues[ns.LockedBlockHash] = true
			}
		}
		return len(lockedValues) > 1
	}
}
