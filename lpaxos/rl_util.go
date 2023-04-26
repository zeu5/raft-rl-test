package lpaxos

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path"

	"github.com/zeu5/raft-rl-test/types"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

type LPaxosDataSet struct {
	savePath     string
	VisitGraph   *types.VisitGraph
	UniqueStates []int
}

func (d *LPaxosDataSet) Save() {
	d.VisitGraph.Record(d.savePath)
}

type LPaxosGraphState struct {
	NodeStates map[uint64]LNodeState
}

func (l *LPaxosGraphState) Hash() string {
	bs, _ := json.Marshal(l.NodeStates)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

func NewLPaxosGraphState(s types.State) *LPaxosGraphState {
	lPaxosState, ok := s.(*LPaxosState)
	if !ok {
		return &LPaxosGraphState{NodeStates: make(map[uint64]LNodeState)}
	}
	return &LPaxosGraphState{NodeStates: lPaxosState.NodeStates}
}

func LPaxosAnalyzer(savePath string) types.Analyzer {
	return func(name string, traces []*types.Trace) types.DataSet {
		dataset := &LPaxosDataSet{
			savePath:     path.Join(savePath, name+".json"),
			VisitGraph:   types.NewVisitGraph(),
			UniqueStates: make([]int, 0),
		}
		uniqueStates := 0
		visitGraph := dataset.VisitGraph
		for _, trace := range traces {
			for i := 0; i < trace.Len(); i++ {
				state, action, nextState, _ := trace.Get(i)
				if visitGraph.Update(NewLPaxosGraphState(state), action.(*LPaxosAction).Hash(), NewLPaxosGraphState(nextState)) {
					uniqueStates += 1
				}
			}
			dataset.UniqueStates = append(dataset.UniqueStates, uniqueStates)
		}
		dataset.Save()
		return dataset
	}
}

func LPaxosComparator(figPath string) types.Comparator {
	return func(names []string, ds []types.DataSet) {
		p := plot.New()
		p.Title.Text = "Comparison"
		p.X.Label.Text = "Iteration"
		p.Y.Label.Text = "States covered"

		for i := 0; i < len(names); i++ {
			dataset := ds[i].(*LPaxosDataSet)
			points := make(plotter.XYs, len(dataset.UniqueStates))
			for i, v := range dataset.UniqueStates {
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
		p.Save(8*vg.Inch, 8*vg.Inch, figPath)
	}
}
