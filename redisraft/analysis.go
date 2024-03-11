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

func NewRedisPartState(s types.State, colors ...RedisRaftColorFunc) *redisRaftState {
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
	// timesteps based version
	AnalyzedSteps  int // count of analyzed steps, used to calculate coverage at fixed intervals
	EpisodeHorizon int // configured episode horizon, used to calculate coverage at fixed intervals

	TracesToAnalyze       []*types.Trace // buffer for traces to analyze
	TimestepsToAnalyze    int            // length of the buffer
	NextIndexOfFirstTrace int            // the first unanalyzed step in the first trace of the buffer
}

func CoverageAnalyzerCtor(horizon int, colors ...RedisRaftColorFunc) func() types.Analyzer {
	return func() types.Analyzer {
		return NewCoverageAnalyzer(horizon, colors...)
	}
}

func NewCoverageAnalyzer(horizon int, colors ...RedisRaftColorFunc) *CoverageAnalyzer {
	return &CoverageAnalyzer{
		colors:          colors,
		uniqueStates:    make(map[string]bool),
		numUniqueStates: make([]int, 0),
		AnalyzedSteps:   0,
		EpisodeHorizon:  horizon,

		TracesToAnalyze:       make([]*types.Trace, 0),
		TimestepsToAnalyze:    0,
		NextIndexOfFirstTrace: 0,
	}
}

// Analyze is called by the benchmark to analyze the traces, it updates the Analyzer's internal buffer of traces to analyze appending the new trace.
// If the total number of timesteps in the traces buffer is than the specified chunk size (horizon, by default), it analyzes <chunk size> timesteps
// and updates the dataset of unique states at each timestep interval. The analyzer keeps track of the last analyzed step and the next index of the first trace to analyze next.
//
// ex. if the horizon is 50, the analyer will analyze traces in chunks of 50 timesteps and will update the dataset of unique states at each 50 timesteps interval.
func (ca *CoverageAnalyzer) Analyze(run int, episode int, startingTimestep int, s string, trace *types.Trace) {
	// fmt.Printf("Analyze Start\n")
	ca.TracesToAnalyze = append(ca.TracesToAnalyze, trace)
	ca.TimestepsToAnalyze += trace.Len()
	// fmt.Printf("Trace added, new length: %d, new timesteps: %d\n", len(ca.TracesToAnalyze), ca.TimestepsToAnalyze)

	if ca.TimestepsToAnalyze >= ca.EpisodeHorizon { // analyze a chunk of size horizon
		// fmt.Printf("Analyzing chunk of size: %d\n", ca.EpisodeHorizon)
		analyzed := 0                      // count of analyzed steps
		index := ca.NextIndexOfFirstTrace  // index of the first unanalyzed step in the first trace of the buffer
		trace := ca.TracesToAnalyze[0]     // the first trace of the buffer
		for analyzed < ca.EpisodeHorizon { // until completing the chunk
			for analyzed < ca.EpisodeHorizon && index < trace.Len() { // until the end of trace or completed the chunk
				// analyze one step
				s, _, _, _ := trace.Get(index)
				sHash := NewRedisPartState(s.(*types.Partition), ca.colors...).Hash()
				if _, ok := ca.uniqueStates[sHash]; !ok {
					ca.uniqueStates[sHash] = true
				}
				analyzed++
				index++
			}
			if index == trace.Len() { // trace ended, remove it from buffer and move to the next one
				// fmt.Printf("Trace ended, analyzed: %d, traces left: %d\n", analyzed, len(ca.TracesToAnalyze))
				if len(ca.TracesToAnalyze) > 1 {
					ca.TracesToAnalyze = ca.TracesToAnalyze[1:]
					trace = ca.TracesToAnalyze[0]
					index = 0
				}
				ca.NextIndexOfFirstTrace = 0
			} else { // analyzed all the steps of the chunk, stop here
				ca.NextIndexOfFirstTrace = index // update the index to start from the next time
			}
		}
		ca.TimestepsToAnalyze -= analyzed                                     // update the total number of timesteps in the buffer
		ca.numUniqueStates = append(ca.numUniqueStates, len(ca.uniqueStates)) // append the number of unique states in the overall list
	}
}

func (ca *CoverageAnalyzer) DataSet() types.DataSet {
	out := make([]int, len(ca.numUniqueStates))
	copy(out, ca.numUniqueStates)
	return out
}

func (ca *CoverageAnalyzer) Reset() {
	// reset the dataset and unique states map
	ca.uniqueStates = make(map[string]bool)
	ca.numUniqueStates = make([]int, 0)

	// reset the buffer of traces to analyze
	ca.AnalyzedSteps = 0
	ca.TracesToAnalyze = make([]*types.Trace, 0)
	ca.TimestepsToAnalyze = 0
	ca.NextIndexOfFirstTrace = 0
}

var _ types.Analyzer = (*CoverageAnalyzer)(nil)

func CoverageComparator(plotPath string, episodeHorizon int) types.Comparator {
	if _, err := os.Stat(plotPath); err != nil {
		os.Mkdir(plotPath, os.ModePerm)
	}
	return func(run, _ int, s []string, ds map[string]types.DataSet) {
		p := plot.New()

		p.Title.Text = "Comparison"
		p.X.Label.Text = "Timesteps"
		p.Y.Label.Text = "States covered"

		coverageData := make(map[string][]int)

		for i := 0; i < len(s); i++ {
			name := s[i]
			dataset := ds[name].([]int)
			coverageData[s[i]] = make([]int, len(dataset))
			copy(coverageData[s[i]], dataset)
			points := make(plotter.XYs, len(dataset))
			for j, v := range dataset {
				points[j] = plotter.XY{
					X: float64(j) * float64(episodeHorizon),
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

func BugCrashAnalyzerCtor(savePath string) func() types.Analyzer {
	return func() types.Analyzer {
		return NewBugCrashAnalyzer(savePath)
	}
}

func NewBugCrashAnalyzer(savePath string) *BugCrashAnalyzer {
	if _, ok := os.Stat(savePath); ok != nil {
		os.MkdirAll(savePath, 0777)
	}
	return &BugCrashAnalyzer{
		savePath:    savePath,
		occurrences: make([]int, 0),
	}
}

func (b *BugCrashAnalyzer) Analyze(run int, episode int, startingTimestep int, s string, trace *types.Trace) {
	// check the trace if any of the nodes have a bug report in stdout or stderr
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
		// if yes, store the occurrence and the logs in a file
		if haveBug {
			b.occurrences = append(b.occurrences, startingTimestep+i)
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
			logFilePath := path.Join(b.savePath, strconv.Itoa(run)+"_"+s+"_ep"+strconv.Itoa(episode)+"_ts"+strconv.Itoa(startingTimestep+i)+"_epStep"+strconv.Itoa(i)+"_bug.log")
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

func BugAnalyzerCtor(savePath string, bugs ...types.BugDesc) func() types.Analyzer {
	return func() types.Analyzer {
		return NewBugAnalyzer(savePath, bugs...)
	}
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

func (ba *BugAnalyzer) Analyze(run int, episode int, startingTimestep int, s string, trace *types.Trace) {
	for _, b := range ba.Bugs {
		_, ok := ba.occurrences[b.Name]
		bugFound, step := b.Check(trace)
		if bugFound { // swapped order just to debug
			if !ok {
				ba.occurrences[b.Name] = make([]int, 0)
			}
			ba.occurrences[b.Name] = append(ba.occurrences[b.Name], startingTimestep+step)
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
	return func(i, _ int, s []string, ds map[string]types.DataSet) {
		for j := 0; j < len(s); j++ {
			name := s[j]
			data := ds[name].([]int)
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
