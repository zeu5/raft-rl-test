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

type CoverageAnalyzer struct {
	colors       []CometColorFunc
	states       map[string]bool
	uniqueStates []int
}

func NewCoverageAnalyzer(colors ...CometColorFunc) *CoverageAnalyzer {
	return &CoverageAnalyzer{
		colors:       colors,
		states:       make(map[string]bool),
		uniqueStates: make([]int, 0),
	}
}

func (c *CoverageAnalyzer) Analyze(run int, episode int, startingTimestep int, s string, trace *types.Trace) {
	for step := 0; step < trace.Len(); step++ {
		s, _, _, _ := trace.Get(step)
		stateHash := cometColoredStateHash(s, c.colors...)
		if _, ok := c.states[stateHash]; !ok {
			c.states[stateHash] = true
		}
	}
	c.uniqueStates = append(c.uniqueStates, len(c.states))
}

func (c *CoverageAnalyzer) DataSet() types.DataSet {
	out := make([]int, len(c.uniqueStates))
	copy(out, c.uniqueStates)
	return out
}

func (c *CoverageAnalyzer) Reset() {
	c.states = make(map[string]bool)
	c.uniqueStates = make([]int, 0)
}

var _ types.Analyzer = &CoverageAnalyzer{}

func CoverageComparator(savePath string) types.Comparator {
	if _, err := os.Stat(savePath); err != nil {
		os.Mkdir(savePath, os.ModePerm)
	}
	return func(run, _ int, s []string, ds map[string]types.DataSet) {
		p := plot.New()

		p.Title.Text = "Comparison"
		p.X.Label.Text = "Iteration"
		p.Y.Label.Text = "States covered"

		coverageData := make(map[string][]int)

		for i := 0; i < len(s); i++ {
			name := s[i]
			dataset := ds[name].([]int)
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

type RecordLogsAnalyzer struct {
	StorePath string
}

func NewRecordLogsAnalyzer(storePath string) *RecordLogsAnalyzer {
	return &RecordLogsAnalyzer{
		StorePath: storePath,
	}
}

func (r *RecordLogsAnalyzer) Analyze(run int, episode int, startingTimestep int, s string, trace *types.Trace) {
	filePath := path.Join(r.StorePath, fmt.Sprintf("%s_%d_%d.log", s, run, episode))
	state, _, _, _ := trace.Last()
	pState := state.(*types.Partition)
	recordLogsToFile(pState, filePath)
}

func (r *RecordLogsAnalyzer) DataSet() types.DataSet {
	return nil
}

func (e *RecordLogsAnalyzer) Reset() {}

var _ types.Analyzer = &RecordLogsAnalyzer{}

func stateToLines(s *types.Partition) []string {
	lines := make([]string, 0)

	for node, rs := range s.ReplicaStates {
		ns := rs.(*CometNodeState)
		_, active := s.ActiveNodes[node]

		if active {
			bs, _ := json.Marshal(ns)
			lines = append(
				lines,
				fmt.Sprintf("NodeID: %d, State: %s", node, string(bs)),
			)
		} else {
			lines = append(lines, fmt.Sprintf("NodeID: %d, State: Inactive", node))
		}
	}
	revMap := make(map[int][]uint64)
	for id, part := range s.PartitionMap {
		list, ok := revMap[part]
		if !ok {
			list = make([]uint64, 0)
		}
		revMap[part] = append(list, id)
	}

	partition := ""
	for _, part := range revMap {
		partition = fmt.Sprintf("%s [", partition)
		for _, replica := range part {
			partition = fmt.Sprintf("%s %d", partition, replica)
		}
		partition = fmt.Sprintf("%s ]", partition)
	}
	partition = fmt.Sprintf("Partition: %s\n", partition)
	lines = append(lines, partition)

	return lines
}

func recordTraceToFile(trace *types.Trace, filePath string) {
	lines := []string{}
	for i := 0; i < trace.Len(); i++ {
		s, a, _, _ := trace.Get(i)
		lines = append(lines, fmt.Sprintf("State: %s, step: %d\n", s.Hash(), i))
		lines = append(lines, stateToLines(s.(*types.Partition))...)
		lines = append(lines, fmt.Sprintf("Action: %s", a.Hash()))
		lines = append(lines, "")
	}

	bs := []byte(strings.Join(lines, "\n"))
	os.WriteFile(filePath, bs, 0644)
}

type RecordStateTraceAnalyzer struct {
	StorePath string
}

func NewRecordStateTraceAnalyzer(storePath string) *RecordStateTraceAnalyzer {
	return &RecordStateTraceAnalyzer{
		StorePath: storePath,
	}
}

func (r *RecordStateTraceAnalyzer) Analyze(run int, episode int, startingTimestep int, s string, trace *types.Trace) {
	filePath := path.Join(r.StorePath, fmt.Sprintf("%s_%d_%d_states.log", s, run, episode))
	recordTraceToFile(trace, filePath)
}

func (r *RecordStateTraceAnalyzer) DataSet() types.DataSet {
	return nil
}

func (e *RecordStateTraceAnalyzer) Reset() {}

var _ types.Analyzer = &RecordStateTraceAnalyzer{}

type CrashesAnalyzer struct {
	StorePath string
}

func NewCrashesAnalyzer(storePath string) *CrashesAnalyzer {
	return &CrashesAnalyzer{
		StorePath: storePath,
	}
}

func (r *CrashesAnalyzer) Analyze(run int, episode int, startingTimestep int, s string, trace *types.Trace) {
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
		stateFilePath := path.Join(r.StorePath, fmt.Sprintf("%s_%d_%d_states.log", s, run, episode))
		logFilePath := path.Join(r.StorePath, fmt.Sprintf("%s_%d_%d.log", s, run, episode))

		s, _, _, _ := trace.Last()
		pState := s.(*types.Partition)

		recordLogsToFile(pState, logFilePath)
		recordTraceToFile(trace, stateFilePath)
	}
}

func (r *CrashesAnalyzer) DataSet() types.DataSet {
	return nil
}

func (e *CrashesAnalyzer) Reset() {}

var _ types.Analyzer = &CrashesAnalyzer{}

type BugLogRecorder struct {
	StorePath string
	Bugs      []types.BugDesc
}

func NewBugLogRecorder(storePath string, bugs ...types.BugDesc) *BugLogRecorder {
	return &BugLogRecorder{
		StorePath: storePath,
		Bugs:      bugs,
	}
}

func (r *BugLogRecorder) Analyze(run int, episode int, startingTimestep int, s string, trace *types.Trace) {
	for _, b := range r.Bugs {
		bugFound, step := b.Check(trace)
		if bugFound { // swapped order just to debug
			bugLogPath := path.Join(r.StorePath, strconv.Itoa(run)+"_"+s+"_"+b.Name+"_"+strconv.Itoa(episode)+"_step"+strconv.Itoa(step)+".log")
			s, _, _, _ := trace.Last()
			pS := s.(*types.Partition)
			recordLogsToFile(pS, bugLogPath)
		}
	}
}

func (r *BugLogRecorder) DataSet() types.DataSet {
	return nil
}

func (e *BugLogRecorder) Reset() {}

var _ types.Analyzer = &BugLogRecorder{}
