package raft

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path"
	"sort"
	"strconv"

	"github.com/zeu5/raft-rl-test/types"
	"go.etcd.io/raft/v3"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

// This file defines the analyzer and comparator for the raft experiments

// The dataset contains a visit graph, unique states observed per iteration
// and the path to save the visit graph at
type RaftDataSet struct {
	savePath     string
	states       map[string]bool
	UniqueStates []int
}

func (d *RaftDataSet) Record() {
	bs, err := json.Marshal(d)
	if err != nil {
		return
	}
	file, err := os.Create(d.savePath)
	if err != nil {
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	writer.Write(bs)
	writer.Flush()

}

type RaftGraphState struct {
	NodeStates map[uint64]raft.Status
}

// Deterministic hash for the visit graph
func (r *RaftGraphState) MarshalJSON() ([]byte, error) {
	marshalStatus := func(s raft.Status) string {
		j := fmt.Sprintf(`{"id":"%x","term":%d,"vote":"%x","commit":%d,"lead":"%x","raftState":%q,"applied":%d,"progress":{`,
			s.ID, s.Term, s.Vote, s.Commit, s.Lead, s.RaftState, s.Applied)

		if len(s.Progress) == 0 {
			j += "},"
		} else {
			keys := make([]int, 0)
			for k := range s.Progress {
				keys = append(keys, int(k))
			}
			sort.Ints(keys)
			for _, k := range keys {
				v := s.Progress[uint64(k)]
				subj := fmt.Sprintf(`"%x":{"match":%d,"next":%d,"state":%q},`, k, v.Match, v.Next, v.State)
				j += subj
			}
			// remove the trailing ","
			j = j[:len(j)-1] + "},"
		}

		j += fmt.Sprintf(`"leadtransferee":"%x"}`, s.LeadTransferee)
		return j
	}

	res := `{"NodeStates":{`
	keys := make([]int, 0)
	for k := range r.NodeStates {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	for _, k := range keys {
		subS := fmt.Sprintf(`"%d":%s,`, k, marshalStatus(r.NodeStates[uint64(k)]))
		res += subS
	}
	res = res[:len(res)-1] + "}}"
	return []byte(res), nil
}

func (r *RaftGraphState) String() string {
	bs, _ := json.Marshal(r)
	return string(bs)
}

func (r *RaftGraphState) Hash() string {
	bs, _ := json.Marshal(r)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

// Analyze the traces to count for unique states (main coverage analyzer)
// Store the resulting visit graph in the specified path/visit_graph_<exp_name>.json
func RaftAnalyzer(savePath string) types.Analyzer {
	if _, err := os.Stat(savePath); err != nil {
		os.Mkdir(savePath, os.ModePerm)
	}
	return func(run int, name string, traces []*types.Trace) types.DataSet {
		dataSet := &RaftDataSet{
			savePath:     path.Join(savePath, strconv.Itoa(run)+"_"+name+".json"),
			states:       make(map[string]bool),
			UniqueStates: make([]int, 0),
		}
		uniqueStates := 0
		for _, trace := range traces {
			for i := 0; i < trace.Len(); i++ {
				state, _, _, _ := trace.Get(i)
				rState, _ := state.(RaftStateType)
				rgState := &RaftGraphState{NodeStates: rState.GetNodeStates()}
				rgStateHash := rgState.Hash()
				if _, ok := dataSet.states[rgStateHash]; !ok {
					dataSet.states[rgStateHash] = true
					uniqueStates += 1
				}
			}
			dataSet.UniqueStates = append(dataSet.UniqueStates, uniqueStates)
		}
		dataSet.Record()
		return dataSet
	}
}

// Print the coverage of the different experiments
func RaftComparator(run int, names []string, datasets []types.DataSet) {
	for i := 0; i < len(names); i++ {
		raftDataSet := datasets[i].(*RaftDataSet)
		fmt.Printf("Coverage for run: %d, experiment: %s, states: %d\n", run, names[i], len(raftDataSet.states))
	}
}

type PlotFilter func([]float64) []float64

func ChainFilters(filters ...PlotFilter) PlotFilter {
	return func(f []float64) []float64 {
		result := f
		for _, f := range filters {
			result = f(result)
		}
		return result
	}
}

func DefaultFilter() PlotFilter {
	return func(f []float64) []float64 {
		return f
	}
}

func Log() PlotFilter {
	return func(f []float64) []float64 {
		res := make([]float64, len(f))
		for i, v := range f {
			res[i] = math.Log(v)
		}
		return res
	}
}

func MinCutOff(min float64) PlotFilter {
	return func(f []float64) []float64 {
		res := make([]float64, 0)
		for _, v := range f {
			if v < min {
				continue
			}
			res = append(res, v)
		}
		return res
	}
}

// Plot coverage of different experiments
func RaftPlotComparator(figPath string) types.Comparator {

	if _, err := os.Stat(figPath); err != nil {
		os.Mkdir(figPath, os.ModePerm)
	}
	return func(run int, names []string, datasets []types.DataSet) {
		p := plot.New()
		p.Title.Text = "Comparison"
		p.X.Label.Text = "Iteration"
		p.Y.Label.Text = "States covered"

		for i := 0; i < len(names); i++ {
			raftDataSet := datasets[i].(*RaftDataSet)
			points := make(plotter.XYs, len(raftDataSet.UniqueStates))
			for i, v := range raftDataSet.UniqueStates {
				points[i] = plotter.XY{
					X: float64(i),
					Y: float64(v),
				}
			}
			line, err := plotter.NewLine(points)
			if err != nil {
				continue
			}
			line.Color = plotutil.Color(i)
			p.Add(line)
			p.Legend.Add(names[i], line)
		}
		p.Save(8*vg.Inch, 8*vg.Inch, path.Join(figPath, strconv.Itoa(run)+"_coverage.png"))
	}
}

var _ types.Comparator = RaftComparator
