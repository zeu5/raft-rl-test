package redisraft

import (
	"strconv"
	"strings"
)

func ParseInfo(info string) *RedisNodeState {
	r := &RedisNodeState{
		Params: make(map[string]string),
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

	return r
}
