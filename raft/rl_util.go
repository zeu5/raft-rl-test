package raft

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image/png"
	"math"
	"os"
	"path"
	"sort"

	"github.com/zeu5/raft-rl-test/types"
	"go.etcd.io/raft/v3"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

// This file defines the analyzer and comparator for the raft experiments

// The dataset contains a visit graph, unique states observed per iteration
// and the path to save the visit graph at
type RaftDataSet struct {
	savePath           string
	visitGraph         *types.VisitGraph
	uniqueStates       []int
	uniqueActualStates []int
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
	return func(name string, traces []*types.Trace) types.DataSet {
		dataSet := &RaftDataSet{
			savePath:           path.Join(savePath, "visit_graph_"+name+".json"),
			visitGraph:         types.NewVisitGraph(),
			uniqueStates:       make([]int, 0),
			uniqueActualStates: make([]int, 0),
		}
		actualStatesMap := make(map[string]bool)
		uniqueStates := 0
		uniqueActualStates := 0
		for _, trace := range traces {
			for i := 0; i < trace.Len(); i++ {
				state, action, nextState, _ := trace.Get(i)
				rState, _ := state.(RaftStateType)
				rNextState, _ := nextState.(RaftStateType)
				rAction, _ := action.(*RaftAction)
				if _, ok := actualStatesMap[state.Hash()]; !ok {
					actualStatesMap[state.Hash()] = true
					uniqueActualStates++
				}
				if dataSet.visitGraph.Update(&RaftGraphState{NodeStates: rState.GetNodeStates()}, rAction.Hash(), &RaftGraphState{NodeStates: rNextState.GetNodeStates()}) {
					uniqueStates++
				}
			}
			dataSet.uniqueStates = append(dataSet.uniqueStates, uniqueStates)
			dataSet.uniqueActualStates = append(dataSet.uniqueActualStates, uniqueActualStates)
		}
		dataSet.visitGraph.Record(path.Join(savePath, "visit_graph_"+name+".json"))
		return dataSet
	}
}

// Print the coverage of the different experiments
func RaftComparator(names []string, datasets []types.DataSet) {
	for i := 0; i < len(names); i++ {
		raftDataSet := datasets[i].(*RaftDataSet)
		fmt.Printf("Coverage for experiment: %s, states: %d\n", names[i], len(raftDataSet.visitGraph.Nodes))
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
func RaftPlotComparator(figPath string, plotFilter PlotFilter) types.Comparator {
	if _, err := os.Stat(figPath); err != nil {
		os.Mkdir(figPath, os.ModePerm)
	}
	return func(names []string, datasets []types.DataSet) {
		p := plot.New()
		p.Title.Text = "Comparison"
		p.X.Label.Text = "Iteration"
		p.Y.Label.Text = "States covered"

		img := vgimg.New(16*vg.Inch, 8*vg.Inch)
		c := draw.New(img)
		cSize := c.Size()
		left, right := draw.Crop(c, 0, c.Min.X-c.Max.X+cSize.X/2, 0, 0), draw.Crop(c, cSize.X/2, 0, 0, 0)

		for i := 0; i < len(names); i++ {
			raftDataSet := datasets[i].(*RaftDataSet)
			points := make(plotter.XYs, len(raftDataSet.uniqueStates))
			for i, v := range raftDataSet.uniqueStates {
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

		p.Draw(left)

		p = plot.New()
		p.Title.Text = "Comparison"
		p.X.Label.Text = "Iteration"
		p.Y.Label.Text = "Actual states covered"

		for i := 0; i < len(names); i++ {
			raftDataSet := datasets[i].(*RaftDataSet)
			points := make(plotter.XYs, len(raftDataSet.uniqueActualStates))
			for i, v := range raftDataSet.uniqueActualStates {
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
		p.Draw(right)

		// if len(names) == 0 {
		// 	return
		// }

		// for i := 0; i < len(names); i++ {
		// 	raftDataSet := datasets[i].(*RaftDataSet)
		// 	points := make([]float64, len(raftDataSet.visitGraph.Nodes))
		// 	j := 0
		// 	for _, v := range raftDataSet.visitGraph.GetVisits() {
		// 		points[j] = float64(v)
		// 		j++
		// 	}
		// 	points = plotFilter(points)
		// 	hist, err := plotter.NewHist(plotter.Values(points), len(points)/10)
		// 	if err != nil {
		// 		continue
		// 	}
		// 	hist.Color = plotutil.Color(i)
		// 	p.Add(hist)
		// 	p.Legend.Add(names[i], hist)
		// }
		// p.Legend.Top = true
		// p.Draw(right)

		f, err := os.Create(path.Join(figPath, "coverage.png"))
		if err != nil {
			return
		}
		png.Encode(f, img.Image())
		f.Close()
	}
}

var _ types.Comparator = RaftComparator
