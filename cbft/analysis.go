package cbft

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/zeu5/raft-rl-test/types"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

func cometColoredStateHash(state types.State, colors ...CometColorFunc) string {
	ps := state.(*types.Partition)
	coloredState := make(map[uint64]map[string]interface{})
	for node, s := range ps.ReplicaStates {
		nodeColoredState := make(map[string]interface{})
		ns := s.(*CometNodeState)
		for _, c := range colors {
			key, val := c(ns)
			nodeColoredState[key] = val
		}
		coloredState[node] = nodeColoredState
	}

	bs, _ := json.Marshal(coloredState)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

func CoverageAnalyzer(colors ...CometColorFunc) types.Analyzer {
	return func(i int, s string, t []*types.Trace) types.DataSet {
		c := make([]int, 0)
		states := make(map[string]bool)
		for _, trace := range t {
			for i := 0; i < trace.Len(); i++ {
				state, _, _, _ := trace.Get(i)
				stateHash := cometColoredStateHash(state, colors...)
				if _, ok := states[stateHash]; !ok {
					states[stateHash] = true
				}
			}
			c = append(c, len(states))
		}
		return c
	}
}

func CoverageComparator(savePath string) types.Comparator {
	if _, err := os.Stat(savePath); err != nil {
		os.Mkdir(savePath, os.ModePerm)
	}
	return func(run int, s []string, ds []types.DataSet) {
		p := plot.New()

		p.Title.Text = "Comparison"
		p.X.Label.Text = "Iteration"
		p.Y.Label.Text = "States covered"

		coverageData := make(map[string][]int)

		for i := 0; i < len(s); i++ {
			dataset := ds[i].([]int)
			coverageData[s[i]] = make([]int, len(dataset))
			copy(coverageData[s[i]], dataset)
			points := make(plotter.XYs, len(dataset))
			for j, v := range dataset {
				points[j] = plotter.XY{
					X: float64(j),
					Y: float64(v),
				}
			}
			line, err := plotter.NewLine(points)
			if err != nil {
				continue
			}
			line.Color = plotutil.Color(i)
			p.Add(line)
			p.Legend.Add(s[i], line)
		}

		p.Save(8*vg.Inch, 8*vg.Inch, path.Join(savePath, strconv.Itoa(run)+"_coverage.png"))

		bs, err := json.Marshal(coverageData)
		if err == nil {
			os.WriteFile(path.Join(savePath, strconv.Itoa(run)+"_data.json"), bs, 0644)
		}
	}
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
