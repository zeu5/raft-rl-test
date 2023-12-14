package explorer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type State struct {
	Key   string
	State map[string]interface{}
}

func (s *State) String() string {
	out := "\n"
	partitionMap := make(map[string]int)
	for k, v := range s.State["PartitionMap"].(map[string]interface{}) {
		partitionMap[k] = int(v.(float64))
	}
	out += fmt.Sprintf("Partition: %s\n", readablePartitionMap(partitionMap))

	activeNodes := make(map[string]bool)
	for k, v := range s.State["ActiveNodes"].(map[string]interface{}) {
		activeNodes[k] = v.(bool)
	}
	nodeColors := make(map[string]map[string]interface{})
	for k, v := range s.State["ReplicaColors"].(map[string]interface{}) {
		params := make(map[string]interface{})
		for pK, pV := range v.(map[string]interface{})["Params"].(map[string]interface{}) {
			params[pK] = pV
		}
		nodeColors[k] = params
	}

	out += "Replica Colors:\n"
	for k, params := range nodeColors {
		paramsS := make([]string, 0)
		for pK, pV := range params {
			paramsS = append(paramsS, fmt.Sprintf("%s:%v", pK, pV))
		}
		sort.Strings(paramsS)
		bs, _ := json.Marshal(params)
		hash := sha256.Sum256(bs)
		colorHash := hex.EncodeToString(hash[:])[0:6]
		out += fmt.Sprintf("Node: %s, Color(%s): %s\n", k, colorHash, strings.Join(paramsS, ","))
	}
	out += fmt.Sprintf("Repeat Count: %d\n", int(s.State["RepeatCount"].(float64)))
	out += fmt.Sprintf("Pending Requests: %d", len(s.State["PendingRequests"].([]interface{})))

	return out
}

func readablePartitionMap(m map[string]int) string {
	revMap := make(map[int][]string)
	for id, part := range m {
		list, ok := revMap[part]
		if !ok {
			list = make([]string, 0)
		}
		revMap[part] = append(list, id)
	}

	result := ""
	for _, part := range revMap {
		result = fmt.Sprintf("%s [", result)
		for _, replica := range part {
			result = fmt.Sprintf("%s %s", result, replica)
		}
		result = fmt.Sprintf("%s ]", result)
	}

	return result
}

type Action struct {
	Key    string
	Action interface{}
}

func (a *Action) String() string {
	aM := a.Action.(map[string]interface{})
	if len(aM) == 0 {
		return a.Key
	}
	partitionMap := make([][]string, 0)
	for i, p := range aM["Partition"].([]interface{}) {
		pA := p.([]interface{})
		if len(pA) == 0 {
			continue
		}
		partitionMap = append(partitionMap, make([]string, 0))
		for _, colorI := range pA {
			c, _ := json.Marshal(colorI.(map[string]interface{})["Params"])
			hash := sha256.Sum256(c)
			color := hex.EncodeToString(hash[:])[0:6]
			partitionMap[i] = append(partitionMap[i], color)
		}
	}
	out := ""
	for _, p := range partitionMap {
		out += " [ "
		for _, c := range p {
			out += (c + " ")
		}
		out += "] "
	}
	return fmt.Sprintf("\nKey: %s\nAction:%s\n", a.Key, out)
}

type Trace struct {
	States     []*State  `json:"states"`
	Actions    []*Action `json:"actions"`
	NextStates []*State  `json:"next_states"`
	Rewards    []bool    `json:"rewards"`
}

func NewTrace() *Trace {
	return &Trace{
		States:     make([]*State, 0),
		Actions:    make([]*Action, 0),
		NextStates: make([]*State, 0),
		Rewards:    make([]bool, 0),
	}
}

func (t *Trace) Len() int {
	return len(t.States)
}

func (t *Trace) Get(index int) (*State, *Action, *State, bool) {
	if index > len(t.States) {
		return nil, nil, nil, false
	}
	return t.States[index], t.Actions[index], t.NextStates[index], t.Rewards[index]
}
