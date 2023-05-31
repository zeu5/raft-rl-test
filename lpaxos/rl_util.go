package lpaxos

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"

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

type LPaxosGraphState struct {
	NodeStates map[uint64]LNodeState
}

func (l *LPaxosGraphState) MarshalJSON() ([]byte, error) {
	keys := make([]int, 0)
	for k := range l.NodeStates {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	res := `{"NodeStates":{`
	if len(keys) == 0 {
		res += ","
	} else {
		for _, k := range keys {
			sub, _ := json.Marshal(l.NodeStates[uint64(k)])
			res += fmt.Sprintf(`"%d":%s,`, k, string(sub))
		}
	}
	res = res[:len(res)-1] + `}}`
	return []byte(res), nil
}

func (l *LPaxosGraphState) Hash() string {
	bs, _ := json.Marshal(l)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

func NewLPaxosGraphState(s types.State) *LPaxosGraphState {
	partition, ok := s.(*types.Partition)
	if ok {
		nodeStates := make(map[uint64]LNodeState)
		for id, s := range partition.ReplicaStates {
			nodeStates[id] = s.(LNodeState)
		}
		return &LPaxosGraphState{NodeStates: copyNodeStates(nodeStates)}
	}

	lPaxosState, ok := s.(*LPaxosState)
	if !ok {
		return &LPaxosGraphState{NodeStates: make(map[uint64]LNodeState)}
	}
	return &LPaxosGraphState{NodeStates: copyNodeStates(lPaxosState.NodeStates)}
}

func LPaxosAnalyzer(savePath string) types.Analyzer {
	if _, err := os.Stat(savePath); err != nil {
		os.Mkdir(savePath, os.ModePerm)
	}
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
				if visitGraph.Update(NewLPaxosGraphState(state), action.Hash(), NewLPaxosGraphState(nextState)) {
					uniqueStates += 1
				}
			}
			dataset.UniqueStates = append(dataset.UniqueStates, uniqueStates)
		}

		dataset.VisitGraph.Record(path.Join(savePath, "visit_graph_"+name+".json"))
		dataset.VisitGraph.Clear()
		return dataset
	}
}

func LPaxosComparator(figPath string) types.Comparator {
	if _, err := os.Stat(figPath); err != nil {
		os.Mkdir(figPath, os.ModePerm)
	}
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
		p.Save(8*vg.Inch, 8*vg.Inch, path.Join(figPath, "coverage.png"))
	}
}

type RewardStatesVisited struct {
	visits     map[string]int
	visitGraph *types.VisitGraph
}

func RewardStatesVisitedAnalyzer(names []string, rewardFuncs []types.RewardFunc, savePath string) types.Analyzer {
	if _, err := os.Stat(savePath); err != nil {
		os.Mkdir(savePath, os.ModePerm)
	}
	return func(s string, traces []*types.Trace) types.DataSet {
		d := RewardStatesVisited{
			visits:     make(map[string]int),
			visitGraph: types.NewVisitGraph(),
		}
		for _, n := range names {
			d.visits[n] = 0
		}
		for _, t := range traces {
			for i := 0; i < t.Len(); i++ {
				s, a, ns, _ := t.Get(i)
				d.visitGraph.Update(NewLPaxosGraphState(s), a.Hash(), NewLPaxosGraphState(ns))
				for j := 0; j < len(names); j++ {
					if rewardFuncs[j](s, ns) {
						d.visits[names[j]]++
					}
				}
			}
		}
		d.visitGraph.Record(path.Join(savePath, "visit_graph_"+s+".json"))
		d.visitGraph.Clear()
		return d
	}
}

func RewardStateComparator() types.Comparator {
	return func(s []string, ds []types.DataSet) {
		for i := 0; i < len(s); i++ {
			fmt.Printf("For experiment: %s\n", s[i])
			rewardvisits := ds[i].(RewardStatesVisited)
			for r, v := range rewardvisits.visits {
				fmt.Printf("Reward visits of state %s: %d\n", r, v)
			}
		}
	}
}
