package types

import (
	"bufio"
	"encoding/json"
	"os"
)

type VisitGraph struct {
	Nodes map[string]*Node
}

func NewVisitGraph() *VisitGraph {
	return &VisitGraph{
		Nodes: make(map[string]*Node),
	}
}

func (v *VisitGraph) Update(from NodeState, action string, to NodeState) bool {
	fromKey := from.Hash()
	toKey := to.Hash()
	new := false
	if _, ok := v.Nodes[fromKey]; !ok {
		v.Nodes[fromKey] = NewNode(from)
		new = true
	}
	if _, ok := v.Nodes[toKey]; !ok {
		v.Nodes[toKey] = NewNode(to)
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

func NewNode(s NodeState) *Node {
	return &Node{
		Key:    s.Hash(),
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
