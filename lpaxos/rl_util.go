package lpaxos

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"

	"github.com/zeu5/raft-rl-test/types"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

type LPaxosDataSet struct {
	states       map[string]bool
	UniqueStates []int
}

func (d *LPaxosDataSet) Record(path string) {
	bs, err := json.Marshal(d)
	if err != nil {
		return
	}
	file, err := os.Create(path)
	if err != nil {
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	writer.Write(bs)
	writer.Flush()
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

type LPaxosAnalyzer struct {
	UniqueStates  []int
	CurExperiment string
	CurRun        int
	states        map[string]bool
	baseSavePath  string
}

func NewLPaxosAnalyzer(savePath string) *LPaxosAnalyzer {
	if _, err := os.Stat(savePath); err != nil {
		os.Mkdir(savePath, os.ModePerm)
	}
	return &LPaxosAnalyzer{
		states:        make(map[string]bool),
		baseSavePath:  savePath,
		CurExperiment: "",
		CurRun:        -1,
		UniqueStates:  make([]int, 0),
	}
}

func (a *LPaxosAnalyzer) Analyze(run int, episode int, startingTimestep int, name string, trace *types.Trace) {
	if a.CurExperiment == "" {
		a.CurExperiment = name
		a.states = make(map[string]bool)
	}
	if a.CurRun == -1 {
		a.CurRun = run
	}
	for i := 0; i < trace.Len(); i++ {
		state, _, _, _ := trace.Get(i)
		gState := NewLPaxosGraphState(state)
		gStateHash := gState.Hash()
		if _, ok := a.states[gStateHash]; !ok {
			a.states[gStateHash] = true
		}
	}
	a.UniqueStates = append(a.UniqueStates, len(a.states))
}

func (a *LPaxosAnalyzer) DataSet() types.DataSet {
	ds := &LPaxosDataSet{
		states:       make(map[string]bool),
		UniqueStates: make([]int, len(a.UniqueStates)),
	}
	for k := range a.states {
		ds.states[k] = true
	}
	copy(ds.UniqueStates, a.UniqueStates)
	ds.Record(path.Join(a.baseSavePath, "run_"+strconv.Itoa(a.CurRun)+"_"+a.CurExperiment+".json"))
	return ds
}

func (a *LPaxosAnalyzer) Reset() {
	a.states = make(map[string]bool)
	a.CurExperiment = ""
	a.CurRun = -1
	a.UniqueStates = make([]int, 0)
}

var _ types.Analyzer = (*LPaxosAnalyzer)(nil)

func LPaxosComparator(figPath string) types.Comparator {
	if _, err := os.Stat(figPath); err != nil {
		os.Mkdir(figPath, os.ModePerm)
	}
	return func(run, _ int, names []string, ds []types.DataSet) {
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
		p.Save(8*vg.Inch, 8*vg.Inch, path.Join(figPath, strconv.Itoa(run)+"_coverage.png"))
	}
}

type RewardStatesVisited struct {
	visits     map[string]int
	visitGraph *types.VisitGraph
}

type RewardStatesVisitedAnalyzer struct {
	names       []string
	rewardFuncs []types.RewardFunc
	savePath    string
	ds          *RewardStatesVisited
	experiment  string
}

func NewRewardStatesVisitedAnalyzer(names []string, rewardFuncs []types.RewardFunc, savePath string) *RewardStatesVisitedAnalyzer {
	if _, err := os.Stat(savePath); err != nil {
		os.Mkdir(savePath, os.ModePerm)
	}
	return &RewardStatesVisitedAnalyzer{
		names:       names,
		rewardFuncs: rewardFuncs,
		savePath:    savePath,
		ds: &RewardStatesVisited{
			visits:     make(map[string]int),
			visitGraph: types.NewVisitGraph(),
		},
		experiment: "",
	}
}

func (ra *RewardStatesVisitedAnalyzer) Analyze(run int, episode int, startingTimestep int, name string, trace *types.Trace) {
	if ra.experiment == "" {
		ra.experiment = name
	}
	for i := 0; i < trace.Len(); i++ {
		s, a, ns, _ := trace.Get(i)
		ra.ds.visitGraph.Update(NewLPaxosGraphState(s), a.Hash(), NewLPaxosGraphState(ns))
		for j := 0; j < len(ra.names); j++ {
			if ra.rewardFuncs[j](s, ns) {
				ra.ds.visits[ra.names[j]]++
			}
		}
	}
}

func (ra *RewardStatesVisitedAnalyzer) DataSet() types.DataSet {
	ra.ds.visitGraph.Record(path.Join(ra.savePath, "visit_graph_"+ra.experiment+".json"))
	ra.ds.visitGraph.Clear()
	return ra.ds
}

func (ra *RewardStatesVisitedAnalyzer) Reset() {
	ra.ds = &RewardStatesVisited{
		visits:     make(map[string]int),
		visitGraph: types.NewVisitGraph(),
	}
	ra.experiment = ""
}

var _ types.Analyzer = (*RewardStatesVisitedAnalyzer)(nil)

func RewardStateComparator() types.Comparator {
	return func(run, _ int, s []string, ds []types.DataSet) {
		for i := 0; i < len(s); i++ {
			fmt.Printf("For experiment: %s\n", s[i])
			rewardvisits := ds[i].(RewardStatesVisited)
			for r, v := range rewardvisits.visits {
				fmt.Printf("Reward visits of state %s: %d\n", r, v)
			}
		}
	}
}

func PaxosStateAbstractor() types.StateAbstractor {
	return func(s types.State) string {
		ls := NewLPaxosGraphState(s)
		return ls.Hash()
	}
}
