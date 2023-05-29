package types

import (
	"bufio"
	"encoding/json"
	"os"
)

// VisitGraph is a graph of the state space that is visited
// nodes indicate states with edges to different states
// Each node also store the number of visits
type VisitGraph struct {
	Nodes map[string]*Node
}

func NewVisitGraph() *VisitGraph {
	return &VisitGraph{
		Nodes: make(map[string]*Node),
	}
}

// Update based on a transition (from, action, to)
// States are indexed by a Hash() function
func (v *VisitGraph) Update(from NodeState, action string, to NodeState) bool {
	fromKey := from.Hash()
	toKey := to.Hash()
	new := false
	if _, ok := v.Nodes[fromKey]; !ok {
		v.Nodes[fromKey] = NewNode(fromKey, from)
		new = true
	}
	if _, ok := v.Nodes[toKey]; !ok {
		v.Nodes[toKey] = NewNode(toKey, to)
		new = true
	}
	v.Nodes[fromKey].Visits += 1
	v.Nodes[fromKey].AddNext(action, toKey)
	v.Nodes[toKey].AddPrev(action, fromKey)
	return new
}

func (v *VisitGraph) GetVisits() map[string]int {
	results := make(map[string]int)
	for k, n := range v.Nodes {
		results[k] = n.Visits
	}
	return results
}

// Record the visit graph as a json into the path specified
func (v *VisitGraph) Record(filePath string) {
	bs, err := json.Marshal(v)
	if err != nil {
		return
	}
	file, err := os.Create(filePath)
	if err != nil {
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	writer.Write(bs)
	writer.Flush()
}

func (v *VisitGraph) Clear() {
	v.Nodes = make(map[string]*Node)
}

type NodeState interface {
	Hash() string
}

type Node struct {
	Key    string
	State  NodeState
	Visits int
	// Next, Prev: Each action can lead to many states
	Next map[string]map[string]bool
	Prev map[string]map[string]bool
}

func NewNode(k string, s NodeState) *Node {
	return &Node{
		Key:    k,
		State:  s,
		Visits: 0,
		Next:   make(map[string]map[string]bool),
		Prev:   make(map[string]map[string]bool),
	}
}

func (n *Node) AddPrev(a, prev string) {
	if _, ok := n.Prev[a]; !ok {
		n.Prev[a] = make(map[string]bool)
	}
	n.Prev[a][prev] = true
}

func (n *Node) AddNext(a, next string) {
	if _, ok := n.Next[a]; !ok {
		n.Next[a] = make(map[string]bool)
	}
	n.Next[a][next] = true
}
