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

type rslPartState struct {
	nodeStates map[uint64]LocalState
}

func (r *rslPartState) Hash() string {
	bs, _ := json.Marshal(r.nodeStates)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

func newRSLPartState(s types.State) *rslPartState {
	p, ok := s.(*types.Partition)
	if ok {
		r := &rslPartState{nodeStates: make(map[uint64]LocalState)}
		for id, s := range p.ReplicaStates {
			r.nodeStates[id] = s.(LocalState).Copy()
		}
		return r
	}
	return &rslPartState{nodeStates: make(map[uint64]LocalState)}
}

func CoverageAnalyzer() types.Analyzer {
	return func(run int, s string, t []*types.Trace) types.DataSet {
		c := make([]int, 0)
		states := make(map[string]bool)
		for _, trace := range t {
			for i := 0; i < trace.Len(); i++ {
				state, _, _, _ := trace.Get(i)
				rslState := newRSLPartState(state)
				rslStateHash := rslState.Hash()
				if _, ok := states[rslStateHash]; !ok {
					states[rslStateHash] = true
				}
			}
			c = append(c, len(states))
		}
		return c
	}
}

func CoverageComparator(plotPath string) types.Comparator {
	if _, err := os.Stat(plotPath); err != nil {
		os.Mkdir(plotPath, os.ModePerm)
	}
	return func(run int, s []string, ds []types.DataSet) {
		p := plot.New()

		p.Title.Text = "Comparison"
		p.X.Label.Text = "Iteration"
		p.Y.Label.Text = "States covered"

		for i := 0; i < len(s); i++ {
			dataset := ds[i].([]int)
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
	}
}
