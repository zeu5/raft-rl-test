package raft

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path"
	"sort"
	"strconv"

	"github.com/zeu5/raft-rl-test/types"
	"go.etcd.io/raft/v3"
	pb "go.etcd.io/raft/v3/raftpb"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

// This file defines the analyzer and comparator for the raft experiments

// The dataset contains unique states observed per iteration
// and the path to save the visit graph at
type RaftDataSet struct {
	savePath     string
	states       map[string]bool // use only keys... it's a set. hash of states are keys - hashmap key: string, val: bool
	UniqueStates []int           // for every iteration, how many unique states seen - len() is the number of iterations (episodes)
}

// stores the RaftDataSet in a file with json format
func (d *RaftDataSet) Record() {
	bs, err := json.Marshal(d)
	if err != nil {
		return
	}
	file, err := os.Create(d.savePath)
	if err != nil {
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	writer.Write(bs)
	writer.Flush()

}

type coloredReplicaState struct {
	Params map[string]interface{} // map string : anyType - this is the abstraction representation.
	// strings are color names and interface{} is the value.
	// The final state abstraction is the concatenation of these.
}

type coloredState struct { // map from specific replica ID to its abstracted state
	NodeStates map[uint64]*coloredReplicaState
}

// hashes the whole system abstracted state
func (c *coloredState) Hash() string { // takes a pointer
	bs, _ := json.Marshal(c)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

// Analyze the traces to count for unique states (main coverage analyzer)
// Store the resulting visit graph in the specified path/visit_graph_<exp_name>.json
// colors ...RaftColorFunc - any number of RaftColorFunc arguments, also zero
func RaftAnalyzer(savePath string, colors ...RaftColorFunc) types.Analyzer {
	if _, err := os.Stat(savePath); err != nil { // make folder if not exist
		os.Mkdir(savePath, os.ModePerm)
	}
	// returns a function
	return func(run int, name string, traces []*types.Trace) types.DataSet {
		dataSet := &RaftDataSet{ // data about one run of one policy/experiment
			savePath:     path.Join(savePath, strconv.Itoa(run)+"_"+name+".json"),
			states:       make(map[string]bool), // set of states
			UniqueStates: make([]int, 0),        // list with cumulative amount of unique states, [i] is after episode i
		}
		uniqueStates := 0              // this grows throughout the whole run
		for _, trace := range traces { // for index, elem is equiv to foreach, can ignore index
			for i := 0; i < trace.Len(); i++ {
				state, _, _, _ := trace.Get(i)                                             // state, action, next_state, reward
				rState, _ := state.(*types.Partition)                                      // .(*types.Partition) type cast into a concrete type - * pointer type - second arg is for safety OK (bool)
				cState := &coloredState{NodeStates: make(map[uint64]*coloredReplicaState)} // & takes reference... constructor of coloredState struct
				for node, s := range rState.ReplicaStates {                                // for each node, take abstracted state
					rcState := &coloredReplicaState{Params: make(map[string]interface{})}
					if _, ok := rState.ActiveNodes[node]; !ok {
						rcState.Params["active"] = false
					} else {
						for _, c := range colors { // fill abstract state for a node
							key, val := c(s.(RaftReplicaState))
							rcState.Params[key] = val
						}
					}
					cState.NodeStates[node] = rcState // put in overall state
				}
				cStateHash := cState.Hash()
				if _, ok := dataSet.states[cStateHash]; !ok { // safe way to query hashmap - if key exists, value is stored in first variable of _,ok
					// executed if key is not in hashmap : !ok
					dataSet.states[cStateHash] = true
					uniqueStates += 1
				}
			}
			dataSet.UniqueStates = append(dataSet.UniqueStates, uniqueStates) // append number of unique states at this episode
		}
		dataSet.Record()
		return dataSet
	}
}

// Print the coverage of the different experiments
func RaftComparator(run int, names []string, datasets []types.DataSet) {
	for i := 0; i < len(names); i++ {
		raftDataSet := datasets[i].(*RaftDataSet)
		fmt.Printf("Coverage for run: %d, experiment: %s, states: %d\n", run, names[i], len(raftDataSet.states))
	}
}

type PlotFilter func([]float64) []float64

func ChainFilters(filters ...PlotFilter) PlotFilter {
	return func(f []float64) []float64 {
		result := f
		for _, f := range filters {
			result = f(result)
		}
		return result
	}
}

func DefaultFilter() PlotFilter {
	return func(f []float64) []float64 {
		return f
	}
}

func Log() PlotFilter {
	return func(f []float64) []float64 {
		res := make([]float64, len(f))
		for i, v := range f {
			res[i] = math.Log(v)
		}
		return res
	}
}

func MinCutOff(min float64) PlotFilter {
	return func(f []float64) []float64 {
		res := make([]float64, 0)
		for _, v := range f {
			if v < min {
				continue
			}
			res = append(res, v)
		}
		return res
	}
}

// Plot coverage of different experiments
func RaftPlotComparator(figPath string) types.Comparator {

	if _, err := os.Stat(figPath); err != nil {
		os.Mkdir(figPath, os.ModePerm)
	}
	return func(run int, names []string, datasets []types.DataSet) {
		p := plot.New()
		p.Title.Text = "Comparison"
		p.X.Label.Text = "Iteration"
		p.Y.Label.Text = "States covered"

		for i := 0; i < len(names); i++ {
			raftDataSet := datasets[i].(*RaftDataSet)
			points := make(plotter.XYs, len(raftDataSet.UniqueStates))
			for i, v := range raftDataSet.UniqueStates {
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

// Plot coverage of different experiments
func RaftEmptyComparator() types.Comparator {

	return func(run int, names []string, datasets []types.DataSet) {

	}
}

var _ types.Comparator = RaftComparator

// NOT USEFUL
type RaftGraphState struct { // useless right now
	NodeStates map[uint64]raft.Status
}

// Deterministic hash for the visit graph
func (r *RaftGraphState) MarshalJSON() ([]byte, error) {
	marshalStatus := func(s raft.Status) string {
		j := fmt.Sprintf(`{"id":"%x","term":%d,"vote":"%x","commit":%d,"lead":"%x","raftState":%q,"applied":%d,"progress":{`,
			s.ID, s.Term, s.Vote, s.Commit, s.Lead, s.RaftState, s.Applied)

		if len(s.Progress) == 0 {
			j += "},"
		} else {
			keys := make([]int, 0)
			for k := range s.Progress {
				keys = append(keys, int(k))
			}
			sort.Ints(keys)
			for _, k := range keys {
				v := s.Progress[uint64(k)]
				subj := fmt.Sprintf(`"%x":{"match":%d,"next":%d,"state":%q},`, k, v.Match, v.Next, v.State)
				j += subj
			}
			// remove the trailing ","
			j = j[:len(j)-1] + "},"
		}

		j += fmt.Sprintf(`"leadtransferee":"%x"}`, s.LeadTransferee)
		return j
	}

	res := `{"NodeStates":{`
	keys := make([]int, 0)
	for k := range r.NodeStates {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	for _, k := range keys {
		subS := fmt.Sprintf(`"%d":%s,`, k, marshalStatus(r.NodeStates[uint64(k)]))
		res += subS
	}
	res = res[:len(res)-1] + "}}"
	return []byte(res), nil
}

func (r *RaftGraphState) String() string {
	bs, _ := json.Marshal(r)
	return string(bs)
}

func (r *RaftGraphState) Hash() string {
	bs, _ := json.Marshal(r)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

func RaftReadableAnalyzer(savePath string) types.Analyzer {
	if _, err := os.Stat(savePath); err != nil { // make folder if not exist
		os.Mkdir(savePath, os.ModePerm)
	}
	saveFolder := path.Join(savePath, "readable")
	os.Mkdir(saveFolder, os.ModePerm)
	// returns a function
	return func(run int, name string, traces []*types.Trace) types.DataSet {
		for j, trace := range traces { // for index, elem is equiv to foreach, can ignore index
			readTrace := make([]string, 0)
			for i := 0; i < trace.Len(); i++ {
				readStep := make([]string, 0)

				readStep = append(readStep, fmt.Sprintf("--- STEP: %d --- \n", i))
				state, action, _, _ := trace.Get(i)   // state, action, next_state, reward
				rState, _ := state.(*types.Partition) // .(*types.Partition) type cast into a concrete type - * pointer type - second arg is for safety OK (bool)

				readState := ReadableState(*rState)
				readStep = append(readStep, readState...)
				readStep = append(readStep, "---------------- \n\n")

				readAction := fmt.Sprintf("ACTION: %s\n\n", action.Hash())
				readStep = append(readStep, readAction)

				readTrace = append(readTrace, readStep...)
			}
			fileName := fmt.Sprintf("%06d.txt", j)
			path := path.Join(saveFolder, fileName)
			uniqueSt := ""
			for _, st := range readTrace {
				uniqueSt = fmt.Sprintf("%s%s", uniqueSt, st)
			}
			os.WriteFile(path, []byte(uniqueSt), 0644)
		}
		return nil
	}
}

// takes a state of the system and returns a list of readable states, one for each replica
func ReadableState(p types.Partition) []string {
	result := make([]string, 0)

	for i := 1; i < len(p.ReplicaStates)+1; i++ { // for each replica state
		result = append(result, ReadableReplicaState(p.ReplicaStates[uint64(i)], uint64(i)))
	}
	result = append(result, "---\n")
	partition := p.PartitionMap
	result = append(result, ReadablePartitionMap(partition))

	return result
}

// formats a replica state in a human-readable form
func ReadableReplicaState(state types.ReplicaState, id uint64) string {
	repState := state.(RaftReplicaState)

	repRaftState := repState.State
	repLog := repState.Log
	filteredLog := filterEntries(repLog)
	strLog := ""
	for _, entry := range filteredLog {
		strLog = fmt.Sprintf("%s, %s", strLog, entry.String())
	}

	softState := repRaftState.BasicStatus.SoftState.RaftState.String()
	Term := repRaftState.BasicStatus.HardState.Term

	s := fmt.Sprintf(" ID: %d | T:%d | S:%s | L:[%s] \n", id, Term, softState, strLog)

	return s
}

// formats a partition map in a human-readable form
func ReadablePartitionMap(m map[uint64]int) string {
	revMap := make(map[int][]uint64)
	for id, part := range m {
		list, ok := revMap[part]
		if !ok {
			list = make([]uint64, 0)
		}
		revMap[part] = append(list, id)
	}

	result := ""
	for _, part := range revMap {
		result = fmt.Sprintf("%s [", result)
		for _, replica := range part {
			result = fmt.Sprintf("%s %d", result, replica)
		}
		result = fmt.Sprintf("%s ]", result)
	}
	result = fmt.Sprintf("%s\n", result)

	return result
}

// filter a log and return only the NormalEntry typed entries (Ignore configuration change entries)
func filterEntries(log []pb.Entry) []pb.Entry {
	result := make([]pb.Entry, 0)
	for _, entry := range log {
		if entry.Type == pb.EntryNormal {
			result = append(result, entry)
		}
	}
	return result
}

// filter a log and return only the NormalEntry typed entries with non-zero length (Ignore configuration change entries)
func filterEntriesNoElection(log []pb.Entry) []pb.Entry {
	result := make([]pb.Entry, 0)
	for _, entry := range log {
		if entry.Type == 0 && len(entry.Data) > 0 {
			result = append(result, entry)
		}
	}
	return result
}
