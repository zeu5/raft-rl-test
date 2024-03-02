package rsl

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path"
	"strconv"

	"github.com/zeu5/raft-rl-test/types"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

type rslColoredState struct {
	Params map[string]interface{}
}

type rslState struct {
	nodeStates map[uint64]*rslColoredState
}

func (r *rslState) Hash() string {
	bs, _ := json.Marshal(r.nodeStates)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

func newRSLPartState(s types.State, colors ...RSLColorFunc) *rslState {
	p, ok := s.(*types.Partition)
	if ok {
		r := &rslState{nodeStates: make(map[uint64]*rslColoredState)}
		for id, s := range p.ReplicaStates {
			color := &rslColoredState{Params: make(map[string]interface{})}
			localState := s.(LocalState).Copy()
			for _, c := range colors {
				k, v := c(localState)
				color.Params[k] = v
			}
			r.nodeStates[id] = color
		}
		return r
	}
	return &rslState{nodeStates: make(map[uint64]*rslColoredState)}
}

type CoverageAnalyzer struct {
	colors          []RSLColorFunc
	uniqueStates    map[string]bool
	numUniqueStates []int
}

func NewCoverageAnalyzer(colors ...RSLColorFunc) *CoverageAnalyzer {
	return &CoverageAnalyzer{
		colors:          colors,
		uniqueStates:    make(map[string]bool),
		numUniqueStates: make([]int, 0),
	}
}

func (ca *CoverageAnalyzer) Analyze(run int, episode int, startingTimestep int, s string, trace *types.Trace) {
	for j := 0; j < trace.Len(); j++ {
		s, _, _, _ := trace.Get(j)
		sHash := newRSLPartState(s, ca.colors...).Hash()
		if _, ok := ca.uniqueStates[sHash]; !ok {
			ca.uniqueStates[sHash] = true
		}
	}
	ca.numUniqueStates = append(ca.numUniqueStates, len(ca.uniqueStates))
}

func (ca *CoverageAnalyzer) DataSet() types.DataSet {
	out := make([]int, len(ca.numUniqueStates))
	copy(out, ca.numUniqueStates)
	return out
}

func (ca *CoverageAnalyzer) Reset() {
	ca.uniqueStates = make(map[string]bool)
	ca.numUniqueStates = make([]int, 0)
}

var _ types.Analyzer = (*CoverageAnalyzer)(nil)

func CoverageComparator(plotPath string) types.Comparator {
	if _, err := os.Stat(plotPath); err != nil {
		os.Mkdir(plotPath, os.ModePerm)
	}
	return func(run, _ int, s []string, ds []types.DataSet) {
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
