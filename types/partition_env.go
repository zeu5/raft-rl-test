package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"sort"
	"strconv"
	"time"

	"github.com/zeu5/raft-rl-test/util"
)

// Color of each replica
type Color interface {
	Hash() string
	Copy() Color
}

type InActiveColor struct{}

func (i *InActiveColor) Hash() string {
	return "Inactive"
}

func (i *InActiveColor) Copy() Color {
	return &InActiveColor{}
}

// Accepts a replica state and colors it
type Painter interface {
	Color(ReplicaState) Color
}

// Generic replica state
type ReplicaState interface{}

// A message that is sent between replicas
type Message interface {
	From() uint64
	To() uint64
	// Key to each message
	Hash() string
}

// The global state of a partitioned system
// To abstract away the details of a particular protocol
type PartitionedSystemState interface {
	// Get the state of a particular replica (indexed by an non-zero integer, starts with 1)
	GetReplicaState(uint64) ReplicaState
	// The messages that are currently in transit
	PendingMessages() map[string]Message
}

// The environment encoding the distributed protocol that can be controlled in a
// partition environment
type PartitionedSystemEnvironment interface {
	// Resets the underlying partitioned environment and returns the initial state
	// Called at the end of each episode
	Reset() PartitionedSystemState
	// Progress the clocks of all replicas by 1
	Tick() PartitionedSystemState
	// Deliver the message and return the resulting state
	DeliverMessage(Message) PartitionedSystemState
	// Drop a message
	DropMessage(Message) PartitionedSystemState
	// Stop a node
	Stop(uint64)
	// Start a node
	Start(uint64)
}

// A state of the partitioned environment
type Partition struct {
	// The colors corresponding to each replica
	ReplicaColors map[uint64]Color
	// The partition that each replica belongs to
	PartitionMap map[uint64]int
	// The underlying state of each replica
	ReplicaStates map[uint64]ReplicaState
	// The actual partition of colors
	// Only this is used to hash (therefore indexed by RL)
	Partition   [][]Color
	RepeatCount int

	// For crashes
	WithCrashes bool
	ActiveNodes map[uint64]bool
}

var _ State = &Partition{}

// Action to transition to a corresponding partition
type CreatePartitionAction struct {
	Partition [][]Color
}

// Action to remain in the given partition
type KeepSamePartitionAction struct{}

func (k *KeepSamePartitionAction) Hash() string {
	return "KeepSamePartition"
}

type StopStartAction struct {
	Node   uint64
	Action string
}

func (s *StopStartAction) Hash() string {
	return s.Action + "_" + strconv.Itoa(int(s.Node))
}

var _ Action = &StopStartAction{}

// Partition slice used to pass to the sort function
type PartitionSlice [][]Color

func (p PartitionSlice) Len() int { return len(p) }

// Two partitions are compared first based on the size and
// then lexicographically based on the colors
func (p PartitionSlice) Less(i, j int) bool {
	if len(p[i]) < len(p[j]) {
		return true
	}
	less := false
	for k := 0; k < len(p[j]); k++ {
		one := p[i][k].Hash()
		two := p[j][k].Hash()
		if one < two {
			less = true
			break
		}
	}
	return less
}
func (p PartitionSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

// ColorSlice used to pass to the sort interface
type ColorSlice []Color

func (c ColorSlice) Len() int { return len(c) }
func (c ColorSlice) Less(i, j int) bool {
	one := c[i].Hash()
	two := c[j].Hash()
	return one < two
}
func (c ColorSlice) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

// implementing the "Action" interface
func (c *CreatePartitionAction) Hash() string {
	bs, _ := json.Marshal(c.Partition)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

// Returns the different possible ways to partition from the current configuration
// Additionally an action to not do anything
func (p *Partition) Actions() []Action {
	colorElems := make([]util.Elem, 0)
	colorsHash := make(map[string]Color)
	for _, c := range p.ReplicaColors {
		hash := c.Hash()
		colorsHash[hash] = c
		colorElems = append(colorElems, util.StringElem(hash))
	}
	partitionActions := make([]Action, 0)
	partitionActions = append(partitionActions, &KeepSamePartitionAction{})
	for _, p := range util.EnumeratePartitions(util.MultiSet(colorElems)) {
		partition := make([][]Color, len(p))
		for i, ps := range p {
			partition[i] = make([]Color, len(ps))
			for j, c := range ps {
				hash := string(c.(util.StringElem))
				partition[i][j] = colorsHash[hash]
			}
		}
		partitionActions = append(partitionActions, &CreatePartitionAction{Partition: partition})
	}
	if p.WithCrashes {
		for node := range p.ReplicaStates {
			if _, ok := p.ActiveNodes[node]; ok {
				partitionActions = append(partitionActions, &StopStartAction{Node: node, Action: "Stop"})
			} else {
				partitionActions = append(partitionActions, &StopStartAction{Node: node, Action: "Start"})
			}
		}
	}
	return partitionActions
}

// hash of a partition state, state representation for the RL agent
func (p *Partition) Hash() string {
	partition := make([][]string, len(p.Partition))
	for i, par := range p.Partition {
		partition[i] = make([]string, len(par))
		for j, color := range par {
			partition[i][j] = color.Hash()
		}
	}
	bs, _ := json.Marshal(map[string]interface{}{"colors": partition, "repeat_count": p.RepeatCount})
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

var _ State = &Partition{}
var _ Action = &CreatePartitionAction{}

func copyPartition(p *Partition) *Partition {
	n := &Partition{
		ReplicaColors: make(map[uint64]Color),
		Partition:     make([][]Color, len(p.Partition)),
		ReplicaStates: make(map[uint64]ReplicaState),
		PartitionMap:  make(map[uint64]int),
		RepeatCount:   p.RepeatCount,
		WithCrashes:   p.WithCrashes,
		ActiveNodes:   make(map[uint64]bool),
	}
	for i, s := range p.ReplicaColors {
		n.ReplicaColors[i] = s.Copy()
	}
	for i, s := range p.ReplicaStates { // if it's a map should we copy?
		n.ReplicaStates[i] = s
	}
	for i, p := range p.PartitionMap {
		n.PartitionMap[i] = p
	}
	for i, par := range p.Partition {
		n.Partition[i] = make([]Color, len(par))
		for j, col := range par {
			n.Partition[i][j] = col.Copy()
		}
	}
	for i := range p.ActiveNodes {
		n.ActiveNodes[i] = p.ActiveNodes[i]
	}
	return n
}

// An environment that encodes ways to partition the replicas
// Implements the "Environment" interface
type PartitionEnv struct {
	NumReplicas   int
	UnderlyingEnv PartitionedSystemEnvironment
	painter       Painter
	CurPartition  *Partition
	messages      map[string]Message
	config        PartitionEnvConfig
	rand          *rand.Rand
}

var _ Environment = &PartitionEnv{}

type PartitionEnvConfig struct {
	Painter                Painter
	Env                    PartitionedSystemEnvironment
	TicketBetweenPartition int
	NumReplicas            int
	MaxMessagesPerTick     int
	StaySameStateUpto      int
	WithCrashes            bool
}

func NewPartitionEnv(c PartitionEnvConfig) *PartitionEnv {
	p := &PartitionEnv{
		NumReplicas:   c.NumReplicas,
		painter:       c.Painter,
		UnderlyingEnv: c.Env,
		messages:      make(map[string]Message),
		CurPartition:  nil,
		config:        c,
		rand:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	p.reset()
	return p
}

func (p *PartitionEnv) reset() {
	s := p.UnderlyingEnv.Reset() // takes the state given by the reset function of the underlying environment
	colors := make([]Color, p.NumReplicas)
	curPartition := &Partition{
		ReplicaColors: make(map[uint64]Color),
		PartitionMap:  make(map[uint64]int),
		ReplicaStates: make(map[uint64]ReplicaState),
		Partition:     make([][]Color, 1),
		RepeatCount:   0,
		WithCrashes:   p.config.WithCrashes,
		ActiveNodes:   make(map[uint64]bool),
	}
	for i := 0; i < p.NumReplicas; i++ {
		id := uint64(i + 1)
		rs := s.GetReplicaState(id)
		color := p.painter.Color(rs)
		colors[i] = color
		curPartition.ReplicaStates[id] = rs
		curPartition.PartitionMap[id] = 0
		curPartition.ReplicaColors[id] = color
		curPartition.ActiveNodes[id] = true
	}
	sort.Sort(ColorSlice(colors))
	curPartition.Partition[0] = colors
	p.CurPartition = curPartition
	p.messages = make(map[string]Message)
	for k, m := range s.PendingMessages() {
		p.messages[k] = m
	}
}

func (p *PartitionEnv) Reset() State {
	p.reset()
	return copyPartition(p.CurPartition)
}

func (p *PartitionEnv) updatePartition(a Action) ([][]uint64, map[uint64]int, bool) {
	newPartition := make([][]uint64, len(p.CurPartition.Partition))
	newPartitionMap := make(map[uint64]int)

	switch a.(type) {
	case *CreatePartitionAction:
	case *KeepSamePartitionAction:
	case *StopStartAction:

	}

	_, notKeepSame := a.(*CreatePartitionAction)

	if notKeepSame {
		ca, _ := a.(*CreatePartitionAction)

		// 1. Change partition
		coloredReplicas := make(map[string][]uint64)
		for r, c := range p.CurPartition.ReplicaColors {
			cHash := c.Hash()
			if _, ok := coloredReplicas[cHash]; !ok {
				coloredReplicas[cHash] = make([]uint64, 0)
			}
			coloredReplicas[cHash] = append(coloredReplicas[cHash], r)
		}
		newPartition = make([][]uint64, len(ca.Partition))
		for i, p := range ca.Partition {
			newPartition[i] = make([]uint64, len(p))
			for j, c := range p {
				cHash := c.Hash()
				nextReplica := coloredReplicas[cHash][0]
				coloredReplicas[cHash] = coloredReplicas[cHash][1:]
				newPartition[i][j] = nextReplica
				newPartitionMap[nextReplica] = i
			}
		}
	} else {
		for i := range p.CurPartition.Partition {
			newPartition[i] = make([]uint64, 0)
		}
		for r, i := range p.CurPartition.PartitionMap {
			newPartition[i] = append(newPartition[i], r)
			newPartitionMap[r] = i
		}
	}
	return newPartition, newPartitionMap, !notKeepSame
}

func (p *PartitionEnv) Step(a Action) State {
	// 1. Change partition
	// 2. Perform ticks, delivering messages in between
	// 3. Update states, partitions and return

	nextState := &Partition{
		ReplicaColors: make(map[uint64]Color),
		PartitionMap:  make(map[uint64]int),
		ReplicaStates: make(map[uint64]ReplicaState),
		Partition:     make([][]Color, 0),
		RepeatCount:   p.CurPartition.RepeatCount,
		WithCrashes:   p.config.WithCrashes,
		ActiveNodes:   make(map[uint64]bool),
	}

	newPartition := make([][]uint64, len(p.CurPartition.Partition))
	newPartitionMap := make(map[uint64]int)
	newActive := make(map[uint64]bool)
	for i := range p.CurPartition.ActiveNodes {
		newActive[i] = true
	}

	ss, isStartStop := a.(*StopStartAction)
	_, isChange := a.(*CreatePartitionAction)
	_, isKeepSame := a.(*KeepSamePartitionAction)

	if isStartStop {
		_, isActive := p.CurPartition.ActiveNodes[ss.Node]
		if ss.Action == "Stop" && isActive {
			p.UnderlyingEnv.Stop(ss.Node)
			delete(newActive, ss.Node)
		} else if ss.Action == "Start" && !isActive {
			p.UnderlyingEnv.Start(ss.Node)
			newActive[ss.Node] = true
		}
	}

	if isChange {
		ca, _ := a.(*CreatePartitionAction)

		// 1. Change partition
		coloredReplicas := make(map[string][]uint64)
		for r, c := range p.CurPartition.ReplicaColors {
			cHash := c.Hash()
			if _, ok := coloredReplicas[cHash]; !ok {
				coloredReplicas[cHash] = make([]uint64, 0)
			}
			coloredReplicas[cHash] = append(coloredReplicas[cHash], r)
		}
		newPartition = make([][]uint64, len(ca.Partition))
		for i, p := range ca.Partition {
			newPartition[i] = make([]uint64, len(p))
			for j, c := range p {
				cHash := c.Hash()
				nextReplica := coloredReplicas[cHash][0]
				coloredReplicas[cHash] = coloredReplicas[cHash][1:]
				newPartition[i][j] = nextReplica
				newPartitionMap[nextReplica] = i
			}
		}
	} else {
		for i := range p.CurPartition.Partition {
			newPartition[i] = make([]uint64, 0)
		}
		for r, i := range p.CurPartition.PartitionMap {
			newPartition[i] = append(newPartition[i], r)
			newPartitionMap[r] = i
		}
	}

	// 2. Perform ticks, delivering messages in between
	var s PartitionedSystemState = nil                     // is this the new state?
	for i := 0; i < p.config.TicketBetweenPartition; i++ { // for the specified number of ticks between two partitions (action of the agent)

		messages := make([]Message, 0)
		if s == nil {
			for _, m := range p.messages { // in the beginning s is nil, use p.messages (stored from previous partition state)
				messages = append(messages, m)
			}
		} else {
			for _, m := range s.PendingMessages() { // these are produced in s at every tick?
				messages = append(messages, m)
			}
		}

		if len(messages) > 0 { // if there are messages to deliver
			p.rand.Shuffle(len(messages), func(i, j int) { // randomize order of messages?
				messages[i], messages[j] = messages[j], messages[i]
			})

			mToDeliver := p.rand.Intn(p.config.MaxMessagesPerTick) // randomly choose how many messages to deliver, up to specified bound
			for j := 0; j < mToDeliver; j++ {
				if len(messages) > 0 {
					next := messages[0] // take the first message
					messages = messages[1:]
					// check if partitioning allows delivery
					fromP, fromOk := newPartitionMap[next.From()]
					toP, toOk := newPartitionMap[next.To()]
					_, toActive := nextState.ActiveNodes[next.To()]
					if (!fromOk || !toOk || fromP == toP) && toActive { // deliver it
						p.UnderlyingEnv.DeliverMessage(next)
					} else { // drop it
						p.UnderlyingEnv.DropMessage(next)
					}
				}
			}
		}
		s = p.UnderlyingEnv.Tick() // make the tick pass on the environment
	}

	// 3. Update states, partitions and return
	p.messages = make(map[string]Message)
	for k, m := range s.PendingMessages() { // save pending messages for next state
		p.messages[k] = m
	}

	// for each replica, get its state from s (underlying env) and store it as it is (rs) + store its abstraction (color)
	for i := 0; i < p.NumReplicas; i++ {
		id := uint64(i + 1)
		rs := s.GetReplicaState(id)
		var color Color = &InActiveColor{}
		if _, isActive := newActive[id]; isActive {
			color = p.painter.Color(rs)
		}
		nextState.ReplicaColors[id] = color // abstraction
		nextState.ReplicaStates[id] = rs    // actual replica state
	}

	// apply partitioning
	nextStatePartition := make([][]Color, len(newPartition))
	for i, p := range newPartition {
		partition := make([]Color, len(p))
		for j, r := range p {
			partition[j] = nextState.ReplicaColors[r]
		}
		sort.Sort(ColorSlice(partition))
		nextStatePartition[i] = partition
	}
	sort.Sort(PartitionSlice(nextStatePartition))
	nextState.Partition = nextStatePartition
	nextState.PartitionMap = newPartitionMap
	nextState.ActiveNodes = newActive
	newRepeatCount := 0
	if isSamePartition(nextState.Partition, p.CurPartition.Partition) && isKeepSame {
		newRepeatCount = p.CurPartition.RepeatCount + 1
		if newRepeatCount > p.config.StaySameStateUpto {
			newRepeatCount = p.config.StaySameStateUpto
		}
	}
	nextState.RepeatCount = newRepeatCount

	p.CurPartition = copyPartition(nextState)

	return nextState
}

// Used for defining a strict policy
// To always pick the "KeepSamePartitionAction"
func PickKeepSame() func(actions []Action) (Action, bool) {
	return func(actions []Action) (Action, bool) {
		for _, a := range actions {
			if _, ok := a.(*KeepSamePartitionAction); ok {
				return a, true
			}
		}
		return nil, false
	}
}

func isSamePartition(one [][]Color, two [][]Color) bool {
	if len(one) != len(two) {
		return false
	}
	for i, p := range one {
		if len(p) != len(two[i]) {
			return false
		}
		for j, v := range p {
			if v.Hash() != two[i][j].Hash() {
				return false
			}
		}
	}
	return true
}
