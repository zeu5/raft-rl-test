package redisraft

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

type redisRaftColoredState struct {
	Params map[string]interface{}
}

type redisRaftState struct {
	nodeStates map[uint64]*redisRaftColoredState
}

func (r *redisRaftState) Hash() string {
	bs, _ := json.Marshal(r.nodeStates)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

func newRedisPartState(s types.State, colors ...RedisRaftColorFunc) *redisRaftState {
	p, ok := s.(*types.Partition)
	if ok {
		r := &redisRaftState{nodeStates: make(map[uint64]*redisRaftColoredState)}
		for id, s := range p.ReplicaStates {
			color := &redisRaftColoredState{Params: make(map[string]interface{})}
			localState := s.(*RedisNodeState).Copy()
			for _, c := range colors {
				k, v := c(localState)
				color.Params[k] = v
			}
			r.nodeStates[id] = color
		}
		return r
	}
	return &redisRaftState{nodeStates: make(map[uint64]*redisRaftColoredState)}
}

func CoverageAnalyzer(colors ...RedisRaftColorFunc) types.Analyzer {
	return func(run int, s string, t []*types.Trace) types.DataSet {
		c := make([]int, 0)
		states := make(map[string]bool)
		for _, trace := range t {
			for i := 0; i < trace.Len(); i++ {
				state, _, _, _ := trace.Get(i)
				redisState := newRedisPartState(state, colors...)
				redisStateHash := redisState.Hash()
				if _, ok := states[redisStateHash]; !ok {
					states[redisStateHash] = true
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
