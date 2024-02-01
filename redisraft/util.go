package redisraft

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/zeu5/raft-rl-test/types"
)

// parser of the info string received from a redis node
func ParseInfo(info string) *RedisNodeState {
	r := &RedisNodeState{
		Params: make(map[string]string),
		Logs:   make([]RedisEntry, 0),
	}

	for _, line := range strings.Split(info, "\r\n") {
		if line == "" || strings.Contains(line, "#") {
			continue
		}
		spl := strings.Split(line, ":")
		if len(spl) < 2 {
			continue
		}

		key := spl[0][5:]
		val := strings.Join(spl[1:], ",")
		r.Params[key] = val
	}

	if role, ok := r.Params["role"]; ok {
		r.State = role
	}
	if termS, ok := r.Params["current_term"]; ok {
		term, err := strconv.Atoi(termS)
		if err == nil {
			r.Term = term
		}
	}
	if commitS, ok := r.Params["commit_index"]; ok {
		commit, err := strconv.Atoi(commitS)
		if err == nil {
			r.Commit = commit
		}
	}
	if appliedS, ok := r.Params["last_applied_index"]; ok {
		applied, err := strconv.Atoi(appliedS)
		if err == nil {
			r.Applied = applied
		}
	}
	if leadS, ok := r.Params["leader_id"]; ok {
		lead, err := strconv.Atoi(leadS)
		if err == nil {
			r.Lead = lead
		}
	}
	if voteS, ok := r.Params["voted_for"]; ok {
		vote, err := strconv.Atoi(voteS)
		if err == nil {
			r.Vote = vote
		}
	}
	if snapS, ok := r.Params["snapshot_last_idx"]; ok {
		snap, err := strconv.Atoi(snapS)
		if err == nil {
			r.Snapshot = snap
		}
	}
	if curIndexS, ok := r.Params["current_index"]; ok {
		curIndex, err := strconv.Atoi(curIndexS)
		if err == nil {
			r.Index = curIndex
		}
	}

	if logEntriesS, ok := r.Params["log_entries"]; ok {
		numEntries, err := strconv.Atoi(logEntriesS)
		if err == nil {
			for i := 0; i < numEntries; i++ {
				eS, ok := r.Params[fmt.Sprintf("entry%d", i)]
				if !ok {
					continue
				}
				newEntry := RedisEntry{
					Index: i,
				}
				for _, keyVal := range strings.Split(eS, ",") {
					splits := strings.Split(keyVal, "=")
					if len(splits) != 2 {
						continue
					}
					key := splits[0]
					val := splits[1]
					switch key {
					case "id":
						newEntry.ID = val
					case "term":
						term, err := strconv.Atoi(val)
						if err == nil {
							newEntry.Term = term
						}
					case "data_len":
						dataLen, err := strconv.Atoi(val)
						if err == nil {
							newEntry.DataLen = dataLen
						}
					case "type":
						entryType, err := strconv.Atoi(val)
						if err == nil {
							newEntry.Type = entryType
						}
					}
				}
				r.Logs = append(r.Logs, newEntry)
			}
		}
	}

	return r
}

// copy a log (list of pb.Entry structs in the raft code)
func copyLog(log []RedisEntry) []RedisEntry {
	newLog := make([]RedisEntry, len(log))
	for i, entry := range log {
		newLog[i] = entry.Copy()
	}

	return newLog
}

func PrintReadableTrace(trace *types.Trace, savePath string) {
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
	// fileName := fmt.Sprintf("%06d.txt", j)
	singleString := ""
	for _, st := range readTrace {
		singleString = fmt.Sprintf("%s%s", singleString, st)
	}
	os.WriteFile(savePath, []byte(singleString), 0644)
}

func ReadableTracePrintable(trace *types.Trace) string {
	readTrace := make([]string, 0)
	for i := 0; i < trace.Len(); i++ {
		readStep := make([]string, 0)

		readStep = append(readStep, fmt.Sprintf("--- STEP: %d --- \n", i))
		state, action, _, _ := trace.Get(i) // state, action, next_state, reward

		// get the hierarchy state of the predicate hierarchy
		predHierState := ""
		addInfo, ok := trace.GetAdditionalInfo(i) // get the additional info for the step
		if ok {
			if addInfo["current_rm_state"] != nil {
				predHierState = addInfo["current_rm_state"].(string) // get the current rm state
			} else {
				predHierState = "nil"
			}
		}

		rState, _ := state.(*types.Partition) // .(*types.Partition) type cast into a concrete type - * pointer type - second arg is for safety OK (bool)

		readState := ReadableState(*rState)
		readStep = append(readStep, readState...)
		readStep = append(readStep, "---------------- \n\n")

		readHierState := fmt.Sprintf("PRED_HIER_STATE: %s\n", predHierState)
		readAction := fmt.Sprintf("ACTION: %s\n\n", action.Hash())

		readStep = append(readStep, readHierState)
		readStep = append(readStep, readAction)

		readTrace = append(readTrace, readStep...)
	}
	// fileName := fmt.Sprintf("%06d.txt", j)
	singleString := ""
	for _, st := range readTrace {
		singleString = fmt.Sprintf("%s%s", singleString, st)
	}

	return singleString
}

// takes a state of the system and returns a list of readable states, one for each replica
func ReadableState(p types.Partition) []string {
	result := make([]string, 0)

	for i := 1; i < len(p.ReplicaStates)+1; i++ { // for each replica state
		if p.ActiveNodes[uint64(i)] {
			result = append(result, ReadableReplicaState(p.ReplicaStates[uint64(i)], uint64(i)))
		} else { // if node is 'inactive'
			result = append(result, fmt.Sprintf(" ID: %d | INACTIVE \n", i))
		}
	}
	result = append(result, "---\n")
	pendReqs := p.PendingRequests
	result = append(result, fmt.Sprintf("PENDING REQUESTS: %d\n", len(pendReqs)))
	result = append(result, "---\n")
	partition := p.PartitionMap
	result = append(result, ReadablePartitionMap(partition))

	return result
}

// formats a replica state in a human-readable form
func ReadableReplicaState(state types.ReplicaState, id uint64) string {
	repState := state.(*RedisNodeState)

	repLog := repState.Logs
	strLog := ""
	for _, entry := range repLog {
		strLog = fmt.Sprintf("%s, %s", strLog, entry.String())
	}

	s := fmt.Sprintf(" ID: %d | T:%d | St:%s | In:%d | Comm:%d | App:%d | Snap:%d | L:[%s] \n",
		id, repState.Term, repState.State, repState.Index, repState.Commit, repState.Applied, repState.Snapshot, strLog)

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
