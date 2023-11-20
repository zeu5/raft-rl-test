package cbft

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/zeu5/raft-rl-test/types"
)

func CoverageAnalyzer(colors ...CometColorFunc) types.Analyzer {
	return func(i int, s string, t []*types.Trace) types.DataSet {
		return nil
	}
}

func CoverageComparator(savePath string) types.Comparator {
	return func(i int, s []string, ds []types.DataSet) {}
}

func RecordLogsAnalyzer(storePath string) types.Analyzer {
	return func(i int, s string, t []*types.Trace) types.DataSet {
		for iter, trace := range t {
			filePath := path.Join(storePath, fmt.Sprintf("%s_%d_%d.log", s, i, iter))
			s, _, _, _ := trace.Last()

			pState := s.(*types.Partition)
			lines := []string{}
			for node, rs := range pState.ReplicaStates {
				cState := rs.(*CometNodeState)

				lines = append(lines, fmt.Sprintf("Logs for node: %d\n\n", node))
				lines = append(lines, "Stdout\n")
				lines = append(lines, cState.LogStdout)
				lines = append(lines, "\nStderr\n")
				lines = append(lines, cState.LogStderr)
			}
			bs := []byte(strings.Join(lines, "\n"))
			os.WriteFile(filePath, bs, 0644)
		}
		return nil
	}
}

func stateToLines(s *types.Partition) []string {
	lines := make([]string, 0)

	for node, s := range s.ReplicaStates {
		ns := s.(*CometNodeState)

		bs, _ := json.Marshal(ns)
		lines = append(
			lines,
			fmt.Sprintf("NodeID: %d, State: %s", node, string(bs)),
		)
	}

	return lines
}

func RecordStateTraceAnalyzer(storePath string) types.Analyzer {
	return func(i int, s string, t []*types.Trace) types.DataSet {
		for iter, trace := range t {
			filePath := path.Join(storePath, fmt.Sprintf("%s_%d_%d_states.log", s, i, iter))

			lines := []string{}
			for i := 0; i < trace.Len(); i++ {
				s, _, _, _ := trace.Get(i)
				lines = append(lines, fmt.Sprintf("State for step: %d\n", i))
				lines = append(lines, stateToLines(s.(*types.Partition))...)
				lines = append(lines, "")
			}

			bs := []byte(strings.Join(lines, "\n"))
			os.WriteFile(filePath, bs, 0644)
		}
		return nil
	}
}
