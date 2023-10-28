package redisraft

import "strings"

func ParseInfo(info string) *RedisNodeState {
	r := &RedisNodeState{
		Params: make(map[string]interface{}),
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
	return r
}
