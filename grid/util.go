package grid

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/zeu5/raft-rl-test/types"
	"gonum.org/v1/plot/plotter"
)

type GridDataSet struct {
	Visits    map[int]map[int]map[int]int
	RLActions map[string]map[string]int
	Height    int
	Width     int
}

type KGrid struct {
	K int
	*GridDataSet
}

var _ plotter.GridXYZ = &KGrid{}

func (g *KGrid) Dims() (int, int) {
	return g.Width, g.Height
}

func (g *KGrid) Z(j, i int) float64 {
	return float64(g.Visits[g.K][i][j])
}

func (g *KGrid) X(j int) float64 {
	return float64(j / 10)
}

func (g *KGrid) Y(i int) float64 {
	return float64(i / 10)
}

func (g *KGrid) Min() float64 {
	return 0.0
}

func (g *KGrid) Max() float64 {
	max := 0
	for _, vals := range g.Visits[g.K] {
		for _, count := range vals {
			if count > max {
				max = count
			}
		}
	}
	return float64(max)
}

func GridAnalyzer(_ int, _ string, traces []*types.Trace) types.DataSet {
	dataSet := &GridDataSet{
		Visits:    make(map[int]map[int]map[int]int),
		RLActions: make(map[string]map[string]int),
		Height:    0,
		Width:     0,
	}
	for _, trace := range traces {
		for i := 0; i < trace.Len(); i++ {
			state, action, _, _ := trace.Get(i)
			stateHash := state.Hash()
			actionHash := action.Hash()
			if _, ok := dataSet.RLActions[stateHash]; !ok {
				dataSet.RLActions[stateHash] = make(map[string]int)
			}
			if _, ok := dataSet.RLActions[stateHash][actionHash]; !ok {
				dataSet.RLActions[stateHash][actionHash] = 0
			}
			dataSet.RLActions[stateHash][actionHash] += 1
			gridPostition := state.(*Position)
			if _, ok := dataSet.Visits[gridPostition.K]; !ok {
				dataSet.Visits[gridPostition.K] = make(map[int]map[int]int)
			}
			if _, ok := dataSet.Visits[gridPostition.K][gridPostition.I]; !ok {
				dataSet.Visits[gridPostition.K][gridPostition.I] = make(map[int]int)
			}
			if _, ok := dataSet.Visits[gridPostition.K][gridPostition.I][gridPostition.J]; !ok {
				dataSet.Visits[gridPostition.K][gridPostition.I][gridPostition.J] = 0
			}
			if gridPostition.I+1 > dataSet.Height {
				dataSet.Height = gridPostition.I + 1
			}
			if gridPostition.J+1 > dataSet.Width {
				dataSet.Width = gridPostition.J + 1
			}
			dataSet.Visits[gridPostition.K][gridPostition.I][gridPostition.J] += 1
		}
	}
	return dataSet
}

func GridPlotComparator(figPath string) types.Comparator {
	return func(run int, s []string, ds []types.DataSet) {
		for i := 0; i < len(s); i++ {
			name := s[i]
			dataSet := ds[i].(*GridDataSet)

			bs, _ := json.Marshal(dataSet)
			os.WriteFile(strconv.Itoa(run)+"_"+name+".json", bs, 0400)

			// p := plot.New()
			// p.Title.Text = name
			// p.Add(plotter.NewHeatMap(dataSet, palette.Heat(20, 0.1)))
			// p.Save(4*vg.Inch, 4*vg.Inch, name+figPath)
		}
	}
}

func GridPositionComparator(iPos, jPos, kPos int) types.Comparator {
	return func(run int, s []string, ds []types.DataSet) {
		for i := 0; i < len(s); i++ {
			name := s[i]
			dataSet := ds[i].(*GridDataSet)

			visits := 0
			_, ok := dataSet.Visits[iPos]
			if ok {
				jV, ok := dataSet.Visits[kPos][iPos][jPos]
				if ok {
					visits = jV
				}
			}

			fmt.Printf("Run: %d, Experiment %s visited %d times\n", run, name, visits)
		}
	}
}

func GridDepthComparator() types.Comparator {
	return func(run int, s []string, ds []types.DataSet) {
		for i := 0; i < len(s); i++ {
			name := s[i]
			dataSet := ds[i].(*GridDataSet)

			gridVisits := make(map[int]int)

			for k, kvs := range dataSet.Visits {
				gridVisits[k] = 0
				for _, ivs := range kvs {
					gridVisits[k] += len(ivs)
				}
			}
			fmt.Printf("Run: %d, Experiment %s:\n", run, name)
			for k, visits := range gridVisits {
				fmt.Printf("Grid %d: covered %d positions\n", k, visits)
			}

			topsa := ""
			topsavisits := -1
			for state, svisits := range dataSet.RLActions {
				for action, visits := range svisits {
					if visits > topsavisits {
						topsa = state + "_" + action
						topsavisits = visits
					}
				}
			}
			fmt.Printf("Top visited state action pair: %s, visits: %d\n", topsa, topsavisits)
		}
	}
}
