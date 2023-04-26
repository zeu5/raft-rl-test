package grid

import (
	"encoding/json"
	"os"

	"github.com/zeu5/raft-rl-test/types"
	"gonum.org/v1/plot/plotter"
)

type GridDataSet struct {
	Visits map[int]map[int]int
	Height int
	Width  int
}

var _ plotter.GridXYZ = &GridDataSet{}

func (g *GridDataSet) Dims() (int, int) {
	return g.Width, g.Height
}

func (g *GridDataSet) Z(j, i int) float64 {
	return float64(g.Visits[i][j])
}

func (g *GridDataSet) X(j int) float64 {
	return float64(j / 10)
}

func (g *GridDataSet) Y(i int) float64 {
	return float64(i / 10)
}

func (g *GridDataSet) Min() float64 {
	return 0.0
}

func (g *GridDataSet) Max() float64 {
	max := 0
	for _, vals := range g.Visits {
		for _, count := range vals {
			if count > max {
				max = count
			}
		}
	}
	return float64(max)
}

func MergeGridDatasets(dataSets []types.DataSet) types.DataSet {
	newDataset := &GridDataSet{
		Visits: make(map[int]map[int]int),
		Height: 0,
		Width:  0,
	}
	for _, d := range dataSets {
		dGrid := d.(*GridDataSet)
		if dGrid.Height > newDataset.Height {
			newDataset.Height = dGrid.Height
		}
		if dGrid.Width > newDataset.Width {
			newDataset.Width = dGrid.Width
		}
		for i, vals := range dGrid.Visits {
			if _, ok := newDataset.Visits[i]; !ok {
				newDataset.Visits[i] = make(map[int]int)
			}
			for j, visits := range vals {
				if _, ok := newDataset.Visits[i][j]; !ok {
					newDataset.Visits[i][j] = 0
				}
				newDataset.Visits[i][j] += visits
			}
		}
	}
	return newDataset
}

func GridAnalyzer(_ string, traces []*types.Trace) types.DataSet {
	dataSet := &GridDataSet{
		Visits: make(map[int]map[int]int),
		Height: 0,
		Width:  0,
	}
	for _, trace := range traces {
		for i := 0; i < trace.Len(); i++ {
			state, _, _, _ := trace.Get(i)
			gridPostition := state.(*Position)
			if _, ok := dataSet.Visits[gridPostition.I]; !ok {
				dataSet.Visits[gridPostition.I] = make(map[int]int)
			}
			if _, ok := dataSet.Visits[gridPostition.I][gridPostition.J]; !ok {
				dataSet.Visits[gridPostition.I][gridPostition.J] = 0
			}
			if gridPostition.I+1 > dataSet.Height {
				dataSet.Height = gridPostition.I + 1
			}
			if gridPostition.J+1 > dataSet.Width {
				dataSet.Width = gridPostition.J + 1
			}
			dataSet.Visits[gridPostition.I][gridPostition.J] += 1
		}
	}
	return dataSet
}

func GridPlotComparator(figPath string) types.Comparator {
	return func(s []string, ds []types.DataSet) {
		for i := 0; i < len(s); i++ {
			name := s[i]
			dataSet := ds[i].(*GridDataSet)

			bs, _ := json.Marshal(dataSet)
			os.WriteFile(name+".json", bs, 0400)

			// p := plot.New()
			// p.Title.Text = name
			// p.Add(plotter.NewHeatMap(dataSet, palette.Heat(20, 0.1)))
			// p.Save(4*vg.Inch, 4*vg.Inch, name+figPath)
		}
	}
}
