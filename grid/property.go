package grid

import "github.com/zeu5/raft-rl-test/types"

// func PosReached(i, j int) *types.Monitor {
// 	monitor := types.NewMonitor()
// 	builder := monitor.Build()
// 	builder.On(InPosition(i, j), "PositionReached").MarkSuccess()
// 	return monitor
// }

// func InPosition(i, j int) types.MonitorCondition {
// 	return func(s types.State, _ types.Action, _ types.State) bool {
// 		position, ok := s.(*Position)
// 		if !ok {
// 			return false
// 		}
// 		return position.I == i && position.J == j
// 	}
// }

func InPosition(i, j, k int) types.RewardFunc {
	return func(s1, s2 types.State) bool {
		pos1, ok := s2.(*Position)
		if !ok {
			return false
		}
		return pos1.I == i && pos1.J == j && pos1.K == k
	}
}

func NextGrid(height, width int) types.RewardFunc {
	return func(s1, s2 types.State) bool {
		pos1, ok := s1.(*Position)
		pos2, _ := s2.(*Position)
		if !ok {
			return false
		}
		return pos1.I == height-1 && pos1.J == width-1 && pos1.K+1 == pos2.K
	}
}

func ToGrid(k int) types.RewardFunc {
	return func(s1, s2 types.State) bool {
		pos1, ok := s1.(*Position)
		pos2, _ := s2.(*Position)
		if !ok {
			return false
		}
		return pos1.K != pos2.K && pos2.K == k
	}
}

func TakesDoor(door Door) types.RewardFunc {
	return func(s1, s2 types.State) bool {
		pos1, ok := s1.(*Position)
		pos2, _ := s2.(*Position)
		if !ok {
			return false
		}
		return pos1.Eq(door.From) && pos2.Eq(door.To)
	}
}
