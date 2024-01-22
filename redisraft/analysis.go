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

type CoverageAnalyzer struct {
	colors          []RedisRaftColorFunc
	uniqueStates    map[string]bool
	numUniqueStates []int
}

func NewCoverageAnalyzer(colors ...RedisRaftColorFunc) *CoverageAnalyzer {
	return &CoverageAnalyzer{
		colors:          colors,
		uniqueStates:    make(map[string]bool),
		numUniqueStates: make([]int, 0),
	}
}

func (ca *CoverageAnalyzer) Analyze(_, run int, s string, trace *types.Trace) {
	for j := 0; j < trace.Len(); j++ {
		s, _, _, _ := trace.Get(j)
		sHash := newRedisPartState(s.(*types.Partition), ca.colors...).Hash()
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

type BugCrashAnalyzer struct {
	savePath    string
	occurrences []int
}

func NewBugCrashAnalyzer(savePath string) *BugCrashAnalyzer {
	return &BugCrashAnalyzer{
		savePath:    savePath,
		occurrences: make([]int, 0),
	}
}

func (b *BugCrashAnalyzer) Analyze(run int, episode int, s string, trace *types.Trace) {
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
				haveBug = true
			}
		}
		if haveBug {
			b.occurrences = append(b.occurrences, episode)
			logs := ""
			for nodeID, s := range pState.ReplicaStates {
				logs += fmt.Sprintf("logs for node: %d\n", nodeID)
				rState := s.(*RedisNodeState)
				logs += "----- Stdout -----\n" + rState.LogStdout + "\n----- Stderr -----\n" + rState.LogStderr + "\n"
				logs += fmt.Sprintf("state for node: %d\n", nodeID)
				bs, err := json.Marshal(rState.Params)
				if err != nil {
					logs += string(bs) + "\n\n"
				}
			}
			logFilePath := path.Join(b.savePath, strconv.Itoa(run)+"_"+s+"_"+strconv.Itoa(i)+"_bug.log")
			os.WriteFile(logFilePath, []byte(logs), 0644)
		}
	}
}

func (b *BugCrashAnalyzer) DataSet() types.DataSet {
	out := make([]int, len(b.occurrences))
	copy(out, b.occurrences)
	return out
}

func (b *BugCrashAnalyzer) Reset() {
	b.occurrences = make([]int, 0)
}

var _ types.Analyzer = (*BugCrashAnalyzer)(nil)

type BugAnalyzer struct {
	Bugs        []types.BugDesc
	occurrences map[string][]int
	savePath    string
}

func NewBugAnalyzer(savePath string, bugs ...types.BugDesc) *BugAnalyzer {
	if _, ok := os.Stat(savePath); ok != nil {
		os.MkdirAll(savePath, 0777)
	}
	return &BugAnalyzer{
		Bugs:        bugs,
		occurrences: make(map[string][]int),
		savePath:    savePath,
	}
}

func (ba *BugAnalyzer) Analyze(run int, episode int, s string, trace *types.Trace) {
	for _, b := range ba.Bugs {
		_, ok := ba.occurrences[b.Name]
		bugFound, step := b.Check(trace)
		if bugFound { // swapped order just to debug
			if !ok {
				ba.occurrences[b.Name] = make([]int, 0)
			}
			ba.occurrences[b.Name] = append(ba.occurrences[b.Name], episode)
			filePrefix := fmt.Sprintf("%d_%s_%s_%d_step%d", run, s, b.Name, episode, step)
			bugPath := path.Join(ba.savePath, filePrefix+"_bug.json")
			trace.Record(bugPath)
			bugReadablePath := path.Join(ba.savePath, filePrefix+"_bug_readable.txt")
			PrintReadableTrace(trace, bugReadablePath)

			bugState, _, _, _ := trace.Get(step)
			pS, okS := bugState.(*types.Partition)
			if okS {
				logs := processLogs(pS)
				logFilePath := path.Join(ba.savePath, filePrefix+"_bug.log")
				os.WriteFile(logFilePath, []byte(logs), 0644)
			}
		}
	}
}

func (ba *BugAnalyzer) DataSet() types.DataSet {
	out := make(map[string][]int)
	for b, i := range ba.occurrences {
		out[b] = make([]int, len(i))
		copy(out[b], i)
	}
	return out
}

func (ba *BugAnalyzer) Reset() {
	ba.occurrences = make(map[string][]int)
}

var _ types.Analyzer = (*BugAnalyzer)(nil)

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
	return func(i, _ int, s []string, ds []types.DataSet) {
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
