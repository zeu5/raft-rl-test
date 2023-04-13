package grid

import "github.com/zeu5/raft-rl-test/types"

func PosReached(i, j int) *types.Monitor {
	monitor := types.NewMonitor()
	builder := monitor.Build()
	builder.On(InPosition(i, j), "PositionReached").MarkSuccess()
	return monitor
}

func InPosition(i, j int) types.MonitorCondition {
	return func(trace *types.Trace) bool {
		last, _, _, ok := trace.Last()
		if !ok {
			return false
		}
		position, ok := last.(*Position)
		if !ok {
			return false
		}
		return position.I == i && position.J == j
	}
}
