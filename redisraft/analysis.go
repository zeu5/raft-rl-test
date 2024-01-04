package redisraft

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

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

func BugAnalyzerCrash(savePath string) types.Analyzer {
	if _, err := os.Stat(savePath); err != nil {
		os.MkdirAll(savePath, os.ModePerm)
	}
	return func(i int, s string, t []*types.Trace) types.DataSet {
		occurrences := make([]int, 0)
		for iter, trace := range t {
			haveBugInTrace := false
			logs := ""

			for i := 0; i < trace.Len(); i++ {
				state, _, _, _ := trace.Get(i)
				pState := state.(*types.Partition)
				haveBug := false
				for _, s := range pState.ReplicaStates {
					rState := s.(*RedisNodeState)
					if rState.LogStdout == "" && rState.LogStderr == "" {
						continue
					}
					stdout := strings.ToLower(rState.LogStdout)
					stderr := strings.ToLower(rState.LogStderr)
					if strings.Contains(stdout, "redis bug report") || strings.Contains(stderr, "redis bug report") {
						haveBugInTrace = true
						haveBug = true
					}
				}
				if haveBug {
					occurrences = append(occurrences, iter)
					logLines := []string{}
					for nodeID, s := range pState.ReplicaStates {
						logLines = append(logLines, fmt.Sprintf("logs for node: %d\n", nodeID))
						rState := s.(*RedisNodeState)
						logLines = append(logLines, "----- Stdout -----", rState.LogStdout, "----- Stderr -----", rState.LogStderr, "\n")
						logLines = append(logLines, fmt.Sprintf("state for node: %d\n", nodeID))
						bs, err := json.Marshal(rState.Params)
						if err != nil {
							logLines = append(logLines, string(bs)+"\n\n")
						}
					}
					logs = strings.Join(logLines, "\n")
				}
			}
			if haveBugInTrace {
				occurrences = append(occurrences, iter)
				logFilePath := path.Join(savePath, s+"_"+strconv.Itoa(i)+"_"+strconv.Itoa(iter)+".log")
				os.WriteFile(logFilePath, []byte(logs), 0644)
			}
		}
		return occurrences
	}
}

// checks all the bugs on the traces
func BugAnalyzer(savePath string, bugs ...types.BugDesc) types.Analyzer {
	if _, ok := os.Stat(savePath); ok == nil {
		os.RemoveAll(savePath)
	}
	os.MkdirAll(savePath, 0777)
	return func(run int, s string, traces []*types.Trace) types.DataSet {
		occurrences := make(map[string][]int)
		for i, t := range traces {
			for _, b := range bugs {
				_, ok := occurrences[b.Name]
				bugFound, step := b.Check(t)
				if bugFound { // swapped order just to debug
					if !ok {
						occurrences[b.Name] = make([]int, 0)
					}
					occurrences[b.Name] = append(occurrences[b.Name], i)
					// save the trace into files
					bugPath := path.Join(savePath, strconv.Itoa(run)+"_"+s+"_"+b.Name+"_"+strconv.Itoa(i)+"_step"+strconv.Itoa(step)+"_bug.json")
					t.Record(bugPath)
					readPath := path.Join(savePath, strconv.Itoa(run)+"_"+s+"_"+b.Name+"_"+strconv.Itoa(i)+"_step"+strconv.Itoa(step)+"_bug_readable.txt")
					PrintReadableTrace(t, readPath)

					// write log file
					bugState, _, _, _ := t.Get(step)
					pS, okS := bugState.(*types.Partition)
					if okS {
						logs := processLogs(pS)
						logFilePath := path.Join(savePath, strconv.Itoa(run)+"_"+s+"_"+b.Name+"_"+strconv.Itoa(i)+"_step"+strconv.Itoa(step)+"_bug.log")
						os.WriteFile(logFilePath, []byte(logs), 0644)
					}
				}
			}
		}
		return occurrences
	}
}

func processLogs(partition *types.Partition) string {
	logLines := []string{}
	for nodeID, s := range partition.ReplicaStates {
		logLines = append(logLines, fmt.Sprintf("logs for node: %d\n", nodeID))
		rState := s.(*RedisNodeState)
		logLines = append(logLines, "----- Stdout -----", rState.LogStdout, "----- Stderr -----", rState.LogStderr, "\n")
		logLines = append(logLines, fmt.Sprintf("state for node: %d\n", nodeID))
		bs, err := json.Marshal(rState.Params)
		if err != nil {
			logLines = append(logLines, string(bs)+"\n\n")
		}
	}
	logs := strings.Join(logLines, "\n")
	return logs
}

func BugComparator() types.Comparator {
	return func(i int, s []string, ds []types.DataSet) {
		for j := 0; j < len(s); j++ {
			data := ds[j].([]int)
			if len(data) == 0 {
				continue
			}
			smallest := data[0]
			for _, o := range data {
				if o < smallest {
					smallest = o
				}
			}
			fmt.Printf("\nFor run: %d, benchmark: %s, first bug occurrence: %d\n", i, s[j], smallest)
		}
	}
}
