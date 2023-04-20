package raft

import (
	"encoding/json"
	"fmt"
	"image/png"
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

func RaftPlotComparator(figPath string) types.Comparator {
	return func(names []string, datasets []types.DataSet) {
		p := plot.New()
		p.Title.Text = "Comparison"
		p.X.Label.Text = "Iteration"
		p.Y.Label.Text = "States covered"

		img := vgimg.New(8*vg.Inch, 4*vg.Inch)
		c := draw.New(img)
		left, right := SplitHorizontal(c, c.Size().X/2)

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

		for i := 0; i < len(names); i++ {
			raftDataSet := datasets[i].(*RaftDataSet)
			numPoints := len(raftDataSet.statesMap)
			points := make(plotter.Values, numPoints)
			j := 0
			for _, v := range raftDataSet.statesMap {
				points[j] = float64(v)
				j++
			}
			hist, err := plotter.NewHist(points, numPoints)
			if err != nil {
				continue
			}
			hist.Color = plotutil.Color(i)
			p.Add(hist)
			p.Legend.Add(names[i], hist)
		}
		p.Draw(right)

		f, err := os.Create(figPath)
		if err != nil {
			return
		}
		png.Encode(f, img.Image())
		f.Close()
	}
}

var _ types.Comparator = RaftComparator

func SplitHorizontal(c draw.Canvas, x vg.Length) (left, right draw.Canvas) {
	return draw.Crop(c, 0, c.Min.X-c.Max.X+x, 0, 0), draw.Crop(c, x, 0, 0, 0)
}
