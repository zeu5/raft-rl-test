package raft

import (
	"encoding/json"
	"fmt"
	"image/png"
	"math"
	"os"

	"github.com/zeu5/raft-rl-test/types"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

type RaftDataSet struct {
	statesMap    map[string]int
	uniqueStates []int
}

func RaftAnalyzer(traces []*types.Trace) types.DataSet {
	dataSet := &RaftDataSet{
		statesMap:    make(map[string]int),
		uniqueStates: make([]int, 0),
	}
	uniqueStates := 0
	for _, trace := range traces {
		for i := 0; i < trace.Len(); i++ {
			state, _, _, _ := trace.Get(i)
			raftState, _ := state.(RaftStateType)
			stateKeyBytes, _ := json.Marshal(raftState.GetNodeStates())
			stateKey := string(stateKeyBytes)
			if _, ok := dataSet.statesMap[stateKey]; !ok {
				dataSet.statesMap[stateKey] = 0
				uniqueStates++
			}
			dataSet.statesMap[stateKey] += 1
		}
		dataSet.uniqueStates = append(dataSet.uniqueStates, uniqueStates)
	}
	return dataSet
}

var _ types.Analyzer = RaftAnalyzer

func RaftComparator(names []string, datasets []types.DataSet) {
	for i := 0; i < len(names); i++ {
		raftDataSet := datasets[i].(*RaftDataSet)
		fmt.Printf("Coverage for experiment: %s, states: %d\n", names[i], len(raftDataSet.statesMap))
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
		first := datasets[0].(*RaftDataSet)
		common := first.statesMap
		all := first.statesMap

		for i := 0; i < len(names); i++ {
			raftDataSet := datasets[i].(*RaftDataSet)
			points := make([]float64, len(raftDataSet.statesMap))
			j := 0
			for state, v := range raftDataSet.statesMap {
				if _, ok := all[state]; !ok {
					all[state] = 0
				}
				points[j] = float64(v)
				j++
			}
			newCommon := make(map[string]int)
			for state := range common {
				if _, ok := raftDataSet.statesMap[state]; ok {
					newCommon[state] = 0
				}
			}
			common = newCommon
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

		diff := make(map[string]int)
		for state := range all {
			if _, ok := common[state]; !ok {
				diff[state] = 0
			}
		}

		fmt.Printf("Number of common states: %d\nNumber of diff states: %d\n", len(common), len(diff))

		f, err := os.Create(figPath)
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
