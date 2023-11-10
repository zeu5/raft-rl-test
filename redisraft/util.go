package redisraft

import (
	"fmt"
	"strconv"
	"strings"
)

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
				newEntry := RedisEntry{}
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
					}
				}
				r.Logs = append(r.Logs, newEntry)
			}
		}
	}

	return r
}
