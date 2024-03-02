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

type GridCoverageAnalyzer struct {
	positions map[int]map[int]map[int]bool
}

func NewGridCoverageAnalyzer() *GridCoverageAnalyzer {
	return &GridCoverageAnalyzer{
		positions: make(map[int]map[int]map[int]bool),
	}
}

func (gca *GridCoverageAnalyzer) Analyze(run int, episode int, startingTimestep int, s string, trace *types.Trace) {
	for j := 0; j < trace.Len(); j++ {
		s, _, _, _ := trace.Get(j)
		gridPosition := s.(*Position)
		if _, ok := gca.positions[gridPosition.K]; !ok {
			gca.positions[gridPosition.K] = make(map[int]map[int]bool)
		}
		if _, ok := gca.positions[gridPosition.K][gridPosition.I]; !ok {
			gca.positions[gridPosition.K][gridPosition.I] = make(map[int]bool)
		}
		gca.positions[gridPosition.K][gridPosition.I][gridPosition.J] = true
	}
}

func (gca *GridCoverageAnalyzer) DataSet() types.DataSet {
	return gca.positions
}

func (gca *GridCoverageAnalyzer) Reset() {
	gca.positions = make(map[int]map[int]map[int]bool)
}

func GridCoverageComparator() types.Comparator {
	return func(i, _ int, s []string, ds []types.DataSet) {
		for i := 0; i < len(s); i++ {
			name := s[i]
			dataSet := ds[i].(map[int]map[int]map[int]bool)
			positions := 0
			for _, g := range dataSet {
				for _, p := range g {
					positions += len(p)
				}
			}

			fmt.Printf("Run: %d, Experiment %s positions covered: %d\n", i, name, positions)
		}
	}
}

type GridAnalyzer struct {
	ds *GridDataSet
}

func NewGridAnalyzer() *GridAnalyzer {
	return &GridAnalyzer{
		ds: &GridDataSet{
			Visits:    make(map[int]map[int]map[int]int),
			RLActions: make(map[string]map[string]int),
			Height:    0,
			Width:     0,
		},
	}
}

func (ga *GridAnalyzer) Analyze(run int, episode int, startingTimestep int, s string, trace *types.Trace) {
	for i := 0; i < trace.Len(); i++ {
		state, action, _, _ := trace.Get(i)
		stateHash := state.Hash()
		actionHash := action.Hash()
		if _, ok := ga.ds.RLActions[stateHash]; !ok {
			ga.ds.RLActions[stateHash] = make(map[string]int)
		}
		if _, ok := ga.ds.RLActions[stateHash][actionHash]; !ok {
			ga.ds.RLActions[stateHash][actionHash] = 0
		}
		ga.ds.RLActions[stateHash][actionHash] += 1
		gridPostition := state.(*Position)
		if _, ok := ga.ds.Visits[gridPostition.K]; !ok {
			ga.ds.Visits[gridPostition.K] = make(map[int]map[int]int)
		}
		if _, ok := ga.ds.Visits[gridPostition.K][gridPostition.I]; !ok {
			ga.ds.Visits[gridPostition.K][gridPostition.I] = make(map[int]int)
		}
		if _, ok := ga.ds.Visits[gridPostition.K][gridPostition.I][gridPostition.J]; !ok {
			ga.ds.Visits[gridPostition.K][gridPostition.I][gridPostition.J] = 0
		}
		if gridPostition.I+1 > ga.ds.Height {
			ga.ds.Height = gridPostition.I + 1
		}
		if gridPostition.J+1 > ga.ds.Width {
			ga.ds.Width = gridPostition.J + 1
		}
		ga.ds.Visits[gridPostition.K][gridPostition.I][gridPostition.J] += 1
	}
}

func (ga *GridAnalyzer) DataSet() types.DataSet {
	return ga.ds
}

func (ga *GridAnalyzer) Reset() {
	ga.ds = &GridDataSet{
		Visits:    make(map[int]map[int]map[int]int),
		RLActions: make(map[string]map[string]int),
		Height:    0,
		Width:     0,
	}
}

var _ types.Analyzer = (*GridAnalyzer)(nil)

func GridPlotComparator(figPath string) types.Comparator {
	return func(run, _ int, s []string, ds []types.DataSet) {
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
	return func(run, _ int, s []string, ds []types.DataSet) {
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
	return func(run, _ int, s []string, ds []types.DataSet) {
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
