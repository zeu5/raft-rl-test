package ratis

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

type ratisRaftColoredState struct {
	Params map[string]interface{}
}

type ratisRaftState struct {
	nodeStates map[uint64]*ratisRaftColoredState
}

func (r *ratisRaftState) Hash() string {
	bs, _ := json.Marshal(r.nodeStates)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

func newRatisPartState(s types.State, colors ...RatisColorFunc) *ratisRaftState {
	p, ok := s.(*types.Partition)
	if ok {
		r := &ratisRaftState{nodeStates: make(map[uint64]*ratisRaftColoredState)}
		for id, s := range p.ReplicaStates {
			color := &ratisRaftColoredState{Params: make(map[string]interface{})}
			localState := s.(*RatisNodeState).Copy()
			for _, c := range colors {
				k, v := c(localState)
				color.Params[k] = v
			}
			r.nodeStates[id] = color
		}
		return r
	}
	return &ratisRaftState{nodeStates: make(map[uint64]*ratisRaftColoredState)}
}

type CoverageAnalyzer struct {
	colors          []RatisColorFunc
	uniqueStates    map[string]bool
	numUniqueStates []int
}

func NewCoverageAnalyzer(colors ...RatisColorFunc) *CoverageAnalyzer {
	return &CoverageAnalyzer{
		colors:          colors,
		uniqueStates:    make(map[string]bool),
		numUniqueStates: make([]int, 0),
	}
}

func (ca *CoverageAnalyzer) Analyze(_, run int, s string, trace *types.Trace) {
	for j := 0; j < trace.Len(); j++ {
		s, _, _, _ := trace.Get(j)
		ratisState := newRatisPartState(s, ca.colors...)
		ratisStateHash := ratisState.Hash()
		if _, ok := ca.uniqueStates[ratisStateHash]; !ok {
			ca.uniqueStates[ratisStateHash] = true
		}
	}
	ca.numUniqueStates = append(ca.numUniqueStates, len(ca.uniqueStates))
}

func (ca *CoverageAnalyzer) DataSet() types.DataSet {
	out := make([]int, len(ca.numUniqueStates))
	copy(out, ca.numUniqueStates)
	return out
}

func (ca *CoverageAnalyzer) Reset() {
	ca.uniqueStates = make(map[string]bool)
	ca.numUniqueStates = make([]int, 0)
}

var _ types.Analyzer = (*CoverageAnalyzer)(nil)

func CoverageComparator(plotPath string) types.Comparator {
	if _, err := os.Stat(plotPath); err != nil {
		os.Mkdir(plotPath, os.ModePerm)
	}
	return func(run, _ int, s []string, ds []types.DataSet) {
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

		p.Save(8*vg.Inch, 8*vg.Inch, path.Join(plotPath, strconv.Itoa(run)+"_coverage.png"))

		bs, err := json.Marshal(coverageData)
		if err == nil {
			os.WriteFile(path.Join(plotPath, strconv.Itoa(run)+"_data.json"), bs, 0644)
		}
	}
}

// func BugAnalyzer(savePath string) types.Analyzer {
// 	if _, err := os.Stat(savePath); err != nil {
// 		os.MkdirAll(savePath, os.ModePerm)
// 	}
// 	return func(i int, s string, t []*types.Trace) types.DataSet {
// 		occurrences := make([]int, 0)
// 		// Todo: analyze to figure out where the bug is

// 		// for iter, trace := range t {
// 		// 	for i := 0; i < trace.Len(); i++ {
// 		// 		state, _, _, _ := trace.Get(i)
// 		// 		pState := state.(*types.Partition)
// 		// 		hasBug := false
// 		// 		if hasBug {
// 		// 			// Record bug
// 		// 		}
// 		// 	}
// 		// }
// 		return occurrences
// 	}
// }

func BugComparator() types.Comparator {
	return func(i, _ int, s []string, ds []types.DataSet) {
		for j := 0; j < len(s); j++ {
			data := ds[j].([]int)
			if len(data) == 0 {
				continue
			}
			smallest := data[0]
			for _, o := range data {
				if o < smallest {
					smallest = o
				}
			}
			fmt.Printf("\nFor run: %d, benchmark: %s, first bug occurrence: %d\n", i, s[j], smallest)
		}
	}
}

type LogAnalyzer struct {
	savePath string
}

func NewLogAnalyzer(savePath string) *LogAnalyzer {
	return &LogAnalyzer{savePath: savePath}
}

func (la *LogAnalyzer) Analyze(run int, episode int, s string, trace *types.Trace) {
	haveBugInTrace := false
	logs := ""

	for i := 0; i < trace.Len(); i++ {
		s, _, _, _ := trace.Get(i)
		pState := s.(*types.Partition)

		logLines := make([]string, 0)
		for nodeID, rs := range pState.ReplicaStates {
			rState := rs.(*RatisNodeState)
			if len(rState.LogStdout) != 0 || len(rState.LogStderr) != 0 {
				logLines = append(logLines, fmt.Sprintf("logs for node: %d\n", nodeID))
				logLines = append(logLines, "----- Stdout -----", rState.LogStdout, "----- Stderr -----", rState.LogStderr, "\n")
			}
		}
		if len(logLines) > 0 {
			logs = strings.Join(logLines, "\n")
			haveBugInTrace = true
		}
	}

	if haveBugInTrace {
		logFilePath := path.Join(la.savePath, strconv.Itoa(run)+"_"+s+"_"+strconv.Itoa(episode)+"_bug.log")
		os.WriteFile(logFilePath, []byte(logs), 0644)
	}
}

func (la *LogAnalyzer) DataSet() types.DataSet {
	return nil
}

func (la *LogAnalyzer) Reset() {
}
