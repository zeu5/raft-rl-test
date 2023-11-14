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

func CoverageAnalyzer(colors ...RatisColorFunc) types.Analyzer {
	return func(run int, s string, t []*types.Trace) types.DataSet {
		c := make([]int, 0)
		states := make(map[string]bool)
		for _, trace := range t {
			for i := 0; i < trace.Len(); i++ {
				state, _, _, _ := trace.Get(i)
				ratisState := newRatisPartState(state, colors...)
				ratisStateHash := ratisState.Hash()
				if _, ok := states[ratisStateHash]; !ok {
					states[ratisStateHash] = true
				}
			}
			c = append(c, len(states))
		}
		return c
	}
}

func CoverageComparator(plotPath string) types.Comparator {
	if _, err := os.Stat(plotPath); err != nil {
		os.Mkdir(plotPath, os.ModePerm)
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

		p.Save(8*vg.Inch, 8*vg.Inch, path.Join(plotPath, strconv.Itoa(run)+"_coverage.png"))

		bs, err := json.Marshal(coverageData)
		if err == nil {
			os.WriteFile(path.Join(plotPath, strconv.Itoa(run)+"_data.json"), bs, 0644)
		}
	}
}

func BugAnalyzer(savePath string) types.Analyzer {
	if _, err := os.Stat(savePath); err != nil {
		os.MkdirAll(savePath, os.ModePerm)
	}
	return func(i int, s string, t []*types.Trace) types.DataSet {
		occurrences := make([]int, 0)
		// Todo: analyze to figure out where the bug is

		// for iter, trace := range t {
		// 	for i := 0; i < trace.Len(); i++ {
		// 		state, _, _, _ := trace.Get(i)
		// 		pState := state.(*types.Partition)
		// 		hasBug := false
		// 		if hasBug {
		// 			// Record bug
		// 		}
		// 	}
		// }
		return occurrences
	}
}

func BugComparator() types.Comparator {
	return func(i int, s []string, ds []types.DataSet) {
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

func LogAnalyzer(savePath string) types.Analyzer {
	return func(i int, s string, t []*types.Trace) types.DataSet {
		for iter, trace := range t {

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
				logFilePath := path.Join(savePath, s+"_"+strconv.Itoa(i)+"_"+strconv.Itoa(iter)+".log")
				os.WriteFile(logFilePath, []byte(logs), 0644)
			}
		}
		return nil
	}
}
