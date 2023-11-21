package cbft

import (
	"strconv"
	"strings"

	"github.com/zeu5/raft-rl-test/types"
)

func AnyReachedRound(round int) types.RewardFuncSingle {
	return func(s types.State) bool {
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			hrs := strings.Split(ns.HeightRoundStep, "/")
			if len(hrs) != 3 {
				continue
			}
			if r, err := strconv.Atoi(hrs[1]); err == nil && r >= round {
				return true
			}
		}
		return false
	}
}

func AllAtleastRound(round int) types.RewardFuncSingle {
	return func(s types.State) bool {
		for _, rs := range s.(*types.Partition).ReplicaStates {
			ns := rs.(*CometNodeState)
			hrs := strings.Split(ns.HeightRoundStep, "/")
			if len(hrs) != 3 {
				return false
			}
			rI, err := strconv.Atoi(hrs[1])
			if err != nil || rI < round {
				return false
			}
		}
		return true
	}
}
