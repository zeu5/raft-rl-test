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

	"github.com/zeu5/raft-rl-test/types"
	"go.etcd.io/raft/v3"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

type RaftDataSet struct {
	savePath     string
	visitGraph   *types.VisitGraph
	uniqueStates []int
}

type RaftGraphState struct {
	NodeStates map[uint64]raft.Status
}

func (r *RaftGraphState) Hash() string {
	bs, _ := json.Marshal(r.NodeStates)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

func RaftAnalyzer(savePath string) types.Analyzer {
	if _, err := os.Stat(savePath); err != nil {
		os.Mkdir(savePath, os.ModePerm)
	}
	return func(name string, traces []*types.Trace) types.DataSet {
		dataSet := &RaftDataSet{
			savePath:     path.Join(savePath, "visit_graph_"+name+".json"),
			visitGraph:   types.NewVisitGraph(),
			uniqueStates: make([]int, 0),
		}
		uniqueStates := 0
		for _, trace := range traces {
			for i := 0; i < trace.Len(); i++ {
				state, action, nextState, _ := trace.Get(i)
				rState, _ := state.(RaftStateType)
				rNextState, _ := nextState.(RaftStateType)
				rAction, _ := action.(*RaftAction)
				if dataSet.visitGraph.Update(&RaftGraphState{NodeStates: rState.GetNodeStates()}, rAction.Hash(), &RaftGraphState{NodeStates: rNextState.GetNodeStates()}) {
					uniqueStates++
				}
			}
			dataSet.uniqueStates = append(dataSet.uniqueStates, uniqueStates)
		}
		dataSet.visitGraph.Record(path.Join(savePath, "visit_graph_"+name+".json"))
		return dataSet
	}
}

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
		left, right := splitHorizontal(c, c.Size().X/2)

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
		p.X.Label.Text = "No of visits"
		p.Y.Label.Text = "Count"

		if len(names) == 0 {
			return
		}

		for i := 0; i < len(names); i++ {
			raftDataSet := datasets[i].(*RaftDataSet)
			points := make([]float64, len(raftDataSet.visitGraph.Nodes))
			j := 0
			for _, v := range raftDataSet.visitGraph.GetVisits() {
				points[j] = float64(v)
				j++
			}
			points = plotFilter(points)
			hist, err := plotter.NewHist(plotter.Values(points), len(points)/10)
			if err != nil {
				continue
			}
			hist.Color = plotutil.Color(i)
			p.Add(hist)
			p.Legend.Add(names[i], hist)
		}
		p.Legend.Top = true
		p.Draw(right)

		f, err := os.Create(path.Join(figPath, "coverage.png"))
		if err != nil {
			return
		}
		png.Encode(f, img.Image())
		f.Close()
	}
}

var _ types.Comparator = RaftComparator

func splitHorizontal(c draw.Canvas, x vg.Length) (left, right draw.Canvas) {
	return draw.Crop(c, 0, c.Min.X-c.Max.X+x, 0, 0), draw.Crop(c, x, 0, 0, 0)
}
