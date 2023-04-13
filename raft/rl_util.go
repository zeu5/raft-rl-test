package raft

import (
	"encoding/json"
	"fmt"

	"github.com/zeu5/raft-rl-test/types"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

type RaftDataSet struct {
	statesMap    map[string]bool
	uniqueStates []int
}

func RaftAnalyzer(traces []*types.Trace) types.DataSet {
	dataSet := &RaftDataSet{
		statesMap:    make(map[string]bool),
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
				dataSet.statesMap[stateKey] = true
				uniqueStates++
			}
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

		datasetPlotter := func(dataset *RaftDataSet) plotter.XYs {
			points := make(plotter.XYs, len(dataset.uniqueStates))
			for i, v := range dataset.uniqueStates {
				points[i] = plotter.XY{
					X: float64(i),
					Y: float64(v),
				}
			}
			return points
		}

		for i := 0; i < len(names); i++ {
			raftDataSet := datasets[i].(*RaftDataSet)
			line, err := plotter.NewLine(datasetPlotter(raftDataSet))
			if err != nil {
				continue
			}
			line.Color = plotutil.Color(i)
			p.Add(line)
			p.Legend.Add(names[i], line)
		}
		p.Save(4*vg.Inch, 4*vg.Inch, figPath)
	}
}

var _ types.Comparator = RaftComparator
