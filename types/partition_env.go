package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"sort"
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

type Request interface{}

// The global state of a partitioned system
// To abstract away the details of a particular protocol
type PartitionedSystemState interface {
	// Get the state of a particular replica (indexed by an non-zero integer, starts with 1)
	GetReplicaState(uint64) ReplicaState
	// The messages that are currently in transit
	PendingMessages() map[string]Message
	// Number of pending requests
	PendingRequests() []Request
	// Can we deliver a request
	CanDeliverRequest() bool
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
	DeliverMessages([]Message) PartitionedSystemState
	// Drop a message
	DropMessages([]Message) PartitionedSystemState
	// Receive a request
	ReceiveRequest(Request) PartitionedSystemState
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
	// About requests
	PendingRequests   []Request
	CanDeliverRequest bool
	// For crashes
	WithCrashes      bool
	RemainingCrashes int
	ActiveNodes      map[uint64]bool
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
	Color  string
	Action string
}

func (s *StopStartAction) Hash() string {
	return s.Action + "_" + s.Color
}

var _ Action = &StopStartAction{}

type SendRequestAction struct {
}

func (s *SendRequestAction) Hash() string {
	return "Request"
}

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
	if p.WithCrashes && p.RemainingCrashes > 0 {
		activeColors := make(map[string]Color)
		haveInactive := false
		for node, color := range p.ReplicaColors {
			if _, ok := p.ActiveNodes[node]; ok {
				activeColors[color.Hash()] = color
			} else {
				haveInactive = true
			}
		}
		for c := range activeColors {
			partitionActions = append(partitionActions, &StopStartAction{Color: c, Action: "Stop"})
		}
		if haveInactive {
			partitionActions = append(partitionActions, &StopStartAction{Action: "Start"})
		}
	}
	if p.CanDeliverRequest && len(p.PendingRequests) > 0 {
		partitionActions = append(partitionActions, &SendRequestAction{})
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
	bs, _ := json.Marshal(map[string]interface{}{"colors": partition, "repeat_count": p.RepeatCount, "pending_requests": len(p.PendingRequests)})
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

var _ State = &Partition{}
var _ Action = &CreatePartitionAction{}

func copyPartition(p *Partition) *Partition {
	n := &Partition{
		ReplicaColors:     make(map[uint64]Color),
		Partition:         make([][]Color, len(p.Partition)),
		ReplicaStates:     make(map[uint64]ReplicaState),
		PartitionMap:      make(map[uint64]int),
		RepeatCount:       p.RepeatCount,
		WithCrashes:       p.WithCrashes,
		RemainingCrashes:  p.RemainingCrashes,
		ActiveNodes:       make(map[uint64]bool),
		PendingRequests:   make([]Request, len(p.PendingRequests)),
		CanDeliverRequest: p.CanDeliverRequest,
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
	copy(n.PendingRequests, p.PendingRequests)
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
	CrashLimit             int
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
		ReplicaColors:     make(map[uint64]Color),
		PartitionMap:      make(map[uint64]int),
		ReplicaStates:     make(map[uint64]ReplicaState),
		Partition:         make([][]Color, 1),
		RepeatCount:       0,
		WithCrashes:       p.config.WithCrashes,
		RemainingCrashes:  p.config.CrashLimit,
		ActiveNodes:       make(map[uint64]bool),
		PendingRequests:   s.PendingRequests(),
		CanDeliverRequest: s.CanDeliverRequest(),
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

// Handle the start and stop action
func (p *PartitionEnv) handleStartStop(a Action) *Partition {
	ss, ok := a.(*StopStartAction)
	if !ok {
		return copyPartition(p.CurPartition)
	}

	// Copy the cur state
	nextState := copyPartition(p.CurPartition)

	newActive := make(map[uint64]bool)
	for i := range p.CurPartition.ActiveNodes {
		newActive[i] = true
	}
	if ss.Action == "Stop" {
		nextState.RemainingCrashes = nextState.RemainingCrashes - 1
		// Stop random node of the color if active
		activeNodes := make([]uint64, 0)
		for node, c := range p.CurPartition.ReplicaColors {
			if c.Hash() == ss.Color {
				activeNodes = append(activeNodes, node)
			}
		}
		if len(activeNodes) > 0 {
			nodeI := p.rand.Intn(len(activeNodes))
			node := activeNodes[nodeI]
			p.UnderlyingEnv.Stop(node)
			delete(newActive, node)
		}
	} else if ss.Action == "Start" {
		// Start a random inactive node
		inactiveNodes := make([]uint64, 0)
		for node := range p.CurPartition.ReplicaColors {
			if _, ok := p.CurPartition.ActiveNodes[node]; !ok {
				inactiveNodes = append(inactiveNodes, node)
			}
		}
		if len(inactiveNodes) > 0 {
			nodeI := p.rand.Intn(len(inactiveNodes))
			node := inactiveNodes[nodeI]
			p.UnderlyingEnv.Start(node)
			newActive[node] = true
		}
	}
	// Update active nodes in Partition
	nextState.ActiveNodes = make(map[uint64]bool)
	for n := range newActive {
		nextState.ActiveNodes[n] = true
	}
	for n, c := range p.CurPartition.ReplicaColors {
		_, inactive := nextState.ActiveNodes[n]
		if inactive {
			nextState.ReplicaColors[n] = &InActiveColor{}
		} else {
			nextState.ReplicaColors[n] = c
		}
	}

	// Recolor based on active nodes and recompute the partition
	newPartition := make([][]uint64, len(p.CurPartition.Partition))
	for i := range p.CurPartition.Partition {
		newPartition[i] = make([]uint64, 0)
	}
	for n, p := range p.CurPartition.PartitionMap {
		newPartition[p] = append(newPartition[p], n)
	}

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
	if isSamePartition(nextStatePartition, p.CurPartition.Partition) {
		nextState.RepeatCount = nextState.RepeatCount + 1
		if nextState.RepeatCount > p.config.StaySameStateUpto {
			nextState.RepeatCount = p.config.StaySameStateUpto
		}
	} else {
		nextState.RepeatCount = 0
	}

	return nextState
}

func (p *PartitionEnv) handlePartition(a Action) *Partition {
	ca, isChange := a.(*CreatePartitionAction)
	_, isKeepSame := a.(*KeepSamePartitionAction)

	if !isChange && !isKeepSame {
		return copyPartition(p.CurPartition)
	}
	nextState := copyPartition(p.CurPartition)
	newPartition := make([][]uint64, len(p.CurPartition.Partition))
	newPartitionMap := make(map[uint64]int)

	if isChange {
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

	var s PartitionedSystemState = nil
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

			mToDeliver := min(len(messages), p.config.MaxMessagesPerTick) // randomly choose how many messages to deliver, up to specified bound
			messagesToDeliver := make([]Message, 0)
			messagesToDrop := make([]Message, 0)
			for j := 0; j < mToDeliver; j++ {
				if len(messages) > 0 {
					next := messages[0] // take the first message
					messages = messages[1:]
					fromP, fromOk := newPartitionMap[next.From()]
					toP, toOk := newPartitionMap[next.To()]
					_, toActive := nextState.ActiveNodes[next.To()]
					// check if partitioning allows delivery
					if (!fromOk || !toOk || fromP == toP) && toActive { // deliver it
						messagesToDeliver = append(messagesToDeliver, next)
					} else { // drop it
						messagesToDrop = append(messagesToDrop, next)
					}

				}
			}
			if len(messagesToDeliver) > 0 {
				p.UnderlyingEnv.DeliverMessages(messagesToDeliver)
			}
			if len(messagesToDrop) > 0 {
				p.UnderlyingEnv.DropMessages(messagesToDrop)
			}
		}
		s = p.UnderlyingEnv.Tick() // make the tick pass on the environment
	}

	p.messages = make(map[string]Message)
	for k, m := range s.PendingMessages() { // save pending messages for next state
		p.messages[k] = m
	}

	for i := 0; i < p.NumReplicas; i++ {
		id := uint64(i + 1)
		rs := s.GetReplicaState(id)
		var color Color = &InActiveColor{}
		if _, isActive := nextState.ActiveNodes[id]; isActive {
			color = p.painter.Color(rs)
		}
		nextState.ReplicaColors[id] = color // abstraction
		nextState.ReplicaStates[id] = rs    // actual replica state
	}

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
	nextState.PendingRequests = s.PendingRequests()
	nextState.CanDeliverRequest = s.CanDeliverRequest()

	newRepeatCount := 0
	if isSamePartition(nextState.Partition, p.CurPartition.Partition) && isKeepSame {
		newRepeatCount = p.CurPartition.RepeatCount + 1
		if newRepeatCount > p.config.StaySameStateUpto {
			newRepeatCount = p.config.StaySameStateUpto
		}
	}
	nextState.RepeatCount = newRepeatCount
	return nextState
}

func (p *PartitionEnv) handleRequest(a Action) *Partition {
	_, isSendRequest := a.(*SendRequestAction)
	if !isSendRequest {
		return copyPartition(p.CurPartition)
	}

	nextState := copyPartition(p.CurPartition)
	if nextState.CanDeliverRequest && len(nextState.PendingRequests) > 0 {
		nextRequest := nextState.PendingRequests[0]
		s := p.UnderlyingEnv.ReceiveRequest(nextRequest)
		nextState.PendingRequests = s.PendingRequests()
		nextState.CanDeliverRequest = s.CanDeliverRequest()

		newPartition := make([][]uint64, len(nextState.Partition))
		for i := range nextState.Partition {
			newPartition[i] = make([]uint64, 0)
		}
		for n, part := range nextState.PartitionMap {
			newPartition[part] = append(newPartition[part], n)
			_, inactive := nextState.ActiveNodes[n]
			rs := s.GetReplicaState(n)
			if inactive {
				nextState.ReplicaColors[n] = &InActiveColor{}
			} else {
				color := p.painter.Color(rs)
				nextState.ReplicaColors[n] = color
			}
			nextState.ReplicaStates[n] = rs
		}

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
		nextState.RepeatCount = 0
		nextState.PendingRequests = s.PendingRequests()
		nextState.CanDeliverRequest = s.CanDeliverRequest()

		return nextState
	}

	return nextState
}

func (p *PartitionEnv) Step(a Action) State {
	// 1. Change partition
	// 2. Perform ticks, delivering messages in between
	// 3. Update states, partitions and return

	var nextState *Partition = nil
	switch a.(type) {
	case *StopStartAction:
		nextState = p.handleStartStop(a)
	case *CreatePartitionAction:
		nextState = p.handlePartition(a)
	case *KeepSamePartitionAction:
		nextState = p.handlePartition(a)
	case *SendRequestAction:
		nextState = p.handleRequest(a)
	}

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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
