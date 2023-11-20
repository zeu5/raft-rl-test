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

func recordLogsToFile(s *types.Partition, file string) {
	lines := []string{}
	for node, rs := range s.ReplicaStates {
		cState := rs.(*CometNodeState)

		lines = append(lines, fmt.Sprintf("Logs for node: %d\n\n", node))
		lines = append(lines, "Stdout\n")
		lines = append(lines, cState.LogStdout)
		lines = append(lines, "\nStderr\n")
		lines = append(lines, cState.LogStderr)
	}
	bs := []byte(strings.Join(lines, "\n"))
	os.WriteFile(file, bs, 0644)
}

func RecordLogsAnalyzer(storePath string) types.Analyzer {
	return func(i int, s string, t []*types.Trace) types.DataSet {
		for iter, trace := range t {
			filePath := path.Join(storePath, fmt.Sprintf("%s_%d_%d.log", s, i, iter))
			s, _, _, _ := trace.Last()

			pState := s.(*types.Partition)
			recordLogsToFile(pState, filePath)
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

func recordTraceToFile(trace *types.Trace, filePath string) {
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

func RecordStateTraceAnalyzer(storePath string) types.Analyzer {
	return func(i int, s string, t []*types.Trace) types.DataSet {
		for iter, trace := range t {
			filePath := path.Join(storePath, fmt.Sprintf("%s_%d_%d_states.log", s, i, iter))
			recordTraceToFile(trace, filePath)
		}
		return nil
	}
}

func CrashesAnalyzer(storePath string) types.Analyzer {
	return func(i int, s string, t []*types.Trace) types.DataSet {
		for iter, trace := range t {
			haveCrash := false
			for step := 0; step < trace.Len(); step++ {
				s, _, _, _ := trace.Get(step)
				ps := s.(*types.Partition)
				for _, rs := range ps.ReplicaStates {
					crs := rs.(*CometNodeState)
					if strings.Contains(crs.LogStdout, "panic") || strings.Contains(crs.LogStderr, "panic") {
						haveCrash = true
					}
				}
				if haveCrash {
					break
				}
			}
			if haveCrash {
				stateFilePath := path.Join(storePath, fmt.Sprintf("%s_%d_%d_states.log", s, i, iter))
				logFilePath := path.Join(storePath, fmt.Sprintf("%s_%d_%d.log", s, i, iter))

				s, _, _, _ := trace.Last()
				pState := s.(*types.Partition)

				recordLogsToFile(pState, logFilePath)
				recordTraceToFile(trace, stateFilePath)
			}
		}
		return nil
	}
}
