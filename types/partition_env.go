package types

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

type InActiveColor struct {
	Color
}

func (i *InActiveColor) Hash() string {
	return "Inactive_" + i.Color.Hash()
}

func (i *InActiveColor) Copy() Color {
	return &InActiveColor{
		Color: i.Color.Copy(),
	}
}

type IntColor struct {
	K string
}

func (i *IntColor) Hash() string {
	return i.K
}

func (i *IntColor) Copy() Color {
	return &IntColor{
		K: i.K,
	}
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

type PartitionedSystemEnvironment interface {
	// Resets the underlying partitioned environment and returns the initial state
	// Called at the end of each episode
	Reset(*EpisodeContext) (PartitionedSystemState, error)
	// Progress the clocks of all replicas by 1
	Tick(*EpisodeContext, int) (PartitionedSystemState, error)
	// Deliver the message and return the resulting state
	DeliverMessages([]Message, *EpisodeContext) (PartitionedSystemState, error)
	// Drop a message
	DropMessages([]Message, *EpisodeContext) (PartitionedSystemState, error)
	// Receive a request
	ReceiveRequest(Request, *EpisodeContext) (PartitionedSystemState, error)
	// Stop a node
	Stop(uint64, *EpisodeContext) error
	// Start a node
	Start(uint64, *EpisodeContext) error
}

type ByzantineEnvironment interface {
	BecomeByzantine(uint64)
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
	MaxInactive      int // max number of simultaneously crashed nodes, 0 means no limit
	ActiveNodes      map[uint64]bool
	// For byzantine actions
	ByzantineNodes     map[uint64]bool
	RemainingByzantine int
	WithByzantine      bool

	// for terminal state
	IsTerminal bool
}

// default terminal partition, state representation for each state out of the specified bounds
func TerminalPartition() *Partition {
	return &Partition{
		IsTerminal: true,
	}
}

func (p *Partition) Terminal() bool {
	return p.IsTerminal
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

type ByzantineAction struct {
	Color string
}

func (b *ByzantineAction) Hash() string {
	return "Byzantine_" + b.Color
}

var _ Action = &ByzantineAction{}

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
	for node, c := range p.ReplicaColors {
		hash := c.Hash()
		colorsHash[hash] = c
		if _, active := p.ActiveNodes[node]; active {
			colorElems = append(colorElems, util.StringElem(hash))
		}
	}
	partitionActions := make([]Action, 0)
	if len(colorElems) > 0 {
		partitionActions = append(partitionActions, &KeepSamePartitionAction{})
		for _, part := range util.EnumeratePartitions(util.MultiSet(colorElems)) {
			partition := make([][]Color, len(part))
			for i, ps := range part {
				partition[i] = make([]Color, len(ps))
				for j, c := range ps {
					hash := string(c.(util.StringElem))
					partition[i][j] = colorsHash[hash]
				}
			}
			inActivePart := make([]Color, 0)
			for n, c := range p.ReplicaColors {
				_, active := p.ActiveNodes[n]
				if !active {
					inActivePart = append(inActivePart, c)
				}
			}
			partition = append(partition, inActivePart)
			partitionActions = append(partitionActions, &CreatePartitionAction{Partition: partition})
		}
	}
	// check for start and stop action
	if p.WithCrashes { // if crashes enabled
		// compute number of inactive nodes
		totalReplicas := len(p.ReplicaColors)
		inactiveReplicas := totalReplicas - len(p.ActiveNodes)

		// stop actions
		if p.RemainingCrashes > 0 && ((inactiveReplicas < p.MaxInactive) || p.MaxInactive == 0) { // crash limits not reached
			activeColors := make(map[string]Color) // list of active colors to eventually crash
			for node, color := range p.ReplicaColors {
				if _, ok := p.ActiveNodes[node]; ok { // if id of the node is in the list of active ones
					activeColors[color.Hash()] = color // add its color to the list
				}
			}

			// add stop actions for the active colors
			for c := range activeColors { // add one stop action for each active node color, will be chosen randomly if multiple nodes with same color
				partitionActions = append(partitionActions, &StopStartAction{Color: c, Action: "Stop"})
			}

		}

		// start action
		if inactiveReplicas > 0 {
			partitionActions = append(partitionActions, &StopStartAction{Action: "Start"}) // add a single start action which will randomly start one inactive node
		}
	}
	// check for request action
	if p.CanDeliverRequest && len(p.PendingRequests) > 0 {
		partitionActions = append(partitionActions, &SendRequestAction{})
	}

	if p.WithByzantine && p.RemainingByzantine > 0 {
		byzantineColors := make(map[string]bool)
		for node, c := range p.ReplicaColors {
			_, byzantine := p.ByzantineNodes[node]
			_, active := p.ActiveNodes[node]
			if !byzantine && active {
				byzantineColors[c.Hash()] = true
			}
		}
		if len(byzantineColors) > 0 {
			for c := range byzantineColors {
				partitionActions = append(partitionActions, &ByzantineAction{Color: c})
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
	out := make(map[string]interface{})
	out["colors"] = partition
	out["repeat_count"] = p.RepeatCount
	out["pending_requests"] = len(p.PendingRequests)
	if p.WithByzantine {
		byzantineColors := make(map[string]bool)
		for node, c := range p.ReplicaColors {
			if _, ok := p.ByzantineNodes[node]; ok {
				byzantineColors[c.Hash()] = true
			}
		}
		out["byzantine_colors"] = byzantineColors
	}

	bs, _ := json.Marshal(out)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

var _ State = &Partition{}
var _ Action = &CreatePartitionAction{}

func copyPartition(p *Partition) *Partition {
	n := &Partition{
		ReplicaColors:      make(map[uint64]Color),
		Partition:          make([][]Color, len(p.Partition)),
		ReplicaStates:      make(map[uint64]ReplicaState),
		PartitionMap:       make(map[uint64]int),
		RepeatCount:        p.RepeatCount,
		WithCrashes:        p.WithCrashes,
		RemainingCrashes:   p.RemainingCrashes,
		MaxInactive:        p.MaxInactive,
		ActiveNodes:        make(map[uint64]bool),
		PendingRequests:    make([]Request, len(p.PendingRequests)),
		CanDeliverRequest:  p.CanDeliverRequest,
		WithByzantine:      p.WithByzantine,
		RemainingByzantine: p.RemainingByzantine,
		ByzantineNodes:     make(map[uint64]bool),
		IsTerminal:         p.IsTerminal,
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
	for node, ok := range p.ByzantineNodes {
		n.ByzantineNodes[node] = ok
	}
	copy(n.PendingRequests, p.PendingRequests)
	return n
}

// An environment that encodes ways to partition the replicas
// Implements the "Environment" interface
type PartitionEnv struct {
	ctx context.Context

	NumReplicas   int
	UnderlyingEnv PartitionedSystemEnvironment
	painter       Painter
	CurPartition  *Partition
	messages      map[string]Message
	config        PartitionEnvConfig
	rand          *rand.Rand

	TerminalPredicate func(*Partition) bool
	UEnvCtor          func(context.Context, PartitionedSystemEnvironmentConfig) PartitionedSystemEnvironment
	UEnvBaseConfig    PartitionedSystemEnvironmentBaseConfig
}

// interfaces for the underlying environment configuration
type PartitionedSystemEnvironmentConfig interface{}     // generic interface for the final config
type PartitionedSystemEnvironmentBaseConfig interface { // base configuration for the underlying environment
	// instantiate into a config using the parallel index given during execution
	Instantiate(int) PartitionedSystemEnvironmentConfig
}

var _ Environment = &PartitionEnv{}

type PartitionEnvConfig struct {
	Ctx context.Context

	Painter                Painter
	Env                    PartitionedSystemEnvironment
	EnvCtor                func(context.Context, PartitionedSystemEnvironmentConfig) PartitionedSystemEnvironment // function to create the underlying environment
	EnvBaseConfig          PartitionedSystemEnvironmentBaseConfig                                                 // base configuration for the underlying environment, instantiate into a config using the parallel index given by the tool
	TicketBetweenPartition int
	NumReplicas            int
	MaxMessagesPerTick     int
	StaySameStateUpto      int
	WithCrashes            bool
	MaxInactive            int // max number of simultaneously crashed nodes, 0 means no limit
	CrashLimit             int
	WithByzantine          bool
	MaxByzantine           int
	RecordStats            bool

	// predicate to check if a state is terminal
	TerminalPredicate            func(*Partition) bool
	TerminalPredicateDescription string
}

func (r *PartitionEnvConfig) Printable() string {
	result := "PartitionEnvConfig: \n"
	result = fmt.Sprintf("%s TicksBetweenPartition: %d\n", result, r.TicketBetweenPartition)
	result = fmt.Sprintf("%s NumReplicas: %d\n", result, r.NumReplicas)
	result = fmt.Sprintf("%s MaxMessagesPerTick: %d\n", result, r.MaxMessagesPerTick)
	result = fmt.Sprintf("%s StaySameStateUpto: %d\n", result, r.StaySameStateUpto)
	result = fmt.Sprintf("%s WithCrashes: %t\n", result, r.WithCrashes)
	result = fmt.Sprintf("%s MaxInactive: %d\n", result, r.MaxInactive)
	result = fmt.Sprintf("%s CrashLimit: %d\n", result, r.CrashLimit)
	result = fmt.Sprintf("%s WithByzantine: %t\n", result, r.WithByzantine)
	result = fmt.Sprintf("%s MaxByzantine: %d\n", result, r.MaxByzantine)
	result = fmt.Sprintf("%s RecordStats: %t\n", result, r.RecordStats)
	result = fmt.Sprintf("%s TerminalPredicateDescription: %s\n", result, r.TerminalPredicateDescription)

	return result
}

func NewPartitionEnv(c PartitionEnvConfig) *PartitionEnv {
	p := &PartitionEnv{
		ctx:           c.Ctx,
		NumReplicas:   c.NumReplicas,
		painter:       c.Painter,
		UnderlyingEnv: c.Env,
		messages:      make(map[string]Message),
		CurPartition:  nil,
		config:        c,
		rand:          rand.New(rand.NewSource(time.Now().UnixNano())),

		TerminalPredicate: c.TerminalPredicate,
		UEnvCtor:          c.EnvCtor,
		UEnvBaseConfig:    c.EnvBaseConfig,
	}
	// p.reset()
	return p
}

func (p *PartitionEnv) SetUp(parallelIndex int) {
	p.UnderlyingEnv = p.UEnvCtor(p.ctx, p.UEnvBaseConfig.Instantiate(parallelIndex))
}

func (p *PartitionEnv) Step(a Action, sCtx *StepContext) (State, error) {
	// 1. Change partition
	// 2. Perform ticks, delivering messages in between
	// 3. Update states, partitions and return
	epCtx := sCtx.EpisodeContext

	start := time.Now()
	var nextState *Partition = nil
	var err error

	epCtx.Report.AddIntEntry(len(p.CurPartition.ActiveNodes), "active_nodes_number", "partitionEnv.StepCtx")

	switch a.(type) {
	case *ByzantineAction:
		nextState, err = p.handleByzantine(a, epCtx)
		epCtx.Report.AddTimeEntry(time.Since(start), "byzantine_action_times", "partitionEnv.StepCtx")
	case *StopStartAction:
		nextState, err = p.handleStartStop(a, epCtx)
		epCtx.Report.AddTimeEntry(time.Since(start), "start_stop_action_times", "partitionEnv.StepCtx")
	case *CreatePartitionAction:
		nextState, err = p.handlePartition(a, epCtx)
		epCtx.Report.AddTimeEntry(time.Since(start), "create_partition_action_times", "partitionEnv.StepCtx")
	case *KeepSamePartitionAction:
		nextState, err = p.handlePartition(a, epCtx)
		epCtx.Report.AddTimeEntry(time.Since(start), "keep_partition_action_times", "partitionEnv.StepCtx")
	case *SendRequestAction:
		nextState, err = p.handleRequest(a, epCtx)
		epCtx.Report.AddTimeEntry(time.Since(start), "request_action_times", "partitionEnv.StepCtx")
	}

	if err != nil {
		return nil, err
	}

	select {
	case <-epCtx.Context.Done():
		return nil, fmt.Errorf("StepCtx : episode timed out")
	default:
	}

	terminal := false
	// check if the state is out of bound
	if p.TerminalPredicate != nil {
		terminal = p.TerminalPredicate(nextState)
	}

	if terminal {
		nextState = TerminalPartition()
	} else {
		p.CurPartition = copyPartition(nextState)
	}

	return nextState, nil
}

func (p *PartitionEnv) handleRequest(a Action, epCtx *EpisodeContext) (*Partition, error) {
	_, isSendRequest := a.(*SendRequestAction)
	if !isSendRequest {
		return copyPartition(p.CurPartition), fmt.Errorf("handleRequestCtx : action is not send request")
	}

	nextState := copyPartition(p.CurPartition)
	if nextState.CanDeliverRequest && len(nextState.PendingRequests) > 0 {
		nextRequest := nextState.PendingRequests[0]
		s, err := p.UnderlyingEnv.ReceiveRequest(nextRequest, epCtx) // call on the underlying environment
		if err != nil {
			return nil, err
		}

		newPartition := make([][]uint64, len(nextState.Partition))
		for i := range nextState.Partition {
			newPartition[i] = make([]uint64, 0)
		}
		for n, part := range nextState.PartitionMap {
			newPartition[part] = append(newPartition[part], n)
			rs := s.GetReplicaState(n)
			color := p.painter.Color(rs)
			if _, ok := nextState.ActiveNodes[n]; !ok {
				color = &InActiveColor{Color: color}
			}
			nextState.ReplicaColors[n] = color
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
		nextState.PendingRequests = s.PendingRequests()     // reducing the number of pending requests is managed in the underlying env
		nextState.CanDeliverRequest = s.CanDeliverRequest() // same for conditions to enable requests delivery

		return nextState, nil
	}

	return nextState, nil
}

func (p *PartitionEnv) handlePartition(a Action, epCtx *EpisodeContext) (*Partition, error) {
	var start time.Time

	ca, isChange := a.(*CreatePartitionAction)
	_, isKeepSame := a.(*KeepSamePartitionAction)

	if !isChange && !isKeepSame {
		return copyPartition(p.CurPartition), fmt.Errorf("handlePartitionCtx : invalid partition action")
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
	var err error

	for i := 0; i < p.config.TicketBetweenPartition; i++ { // for the specified number of ticks between two partitions (action of the agent)
		tickStartTime := time.Now()
		select {
		case <-epCtx.Context.Done():
			return nil, fmt.Errorf("handlePartitionCtx : episode timed out")
		default:
		}
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

		// add current messages to report
		for _, m := range messages {
			messageString := fmt.Sprintf("tick: %d | from: %d | to: %d | %s", i, m.From(), m.To(), m.Hash())
			epCtx.Report.AddStringEntry(messageString, "messages_available", "partitionEnv.handlePartitionCtx")
		}

		mToDeliver := 0

		epCtx.Report.AddIntEntry(len(messages), "messages_total_number", "partitionEnv.handlePartitionCtx")

		if len(messages) > 0 { // if there are messages to deliver
			p.rand.Shuffle(len(messages), func(i, j int) { // randomize order of messages?
				messages[i], messages[j] = messages[j], messages[i]
			})

			mToDeliver = min(len(messages), p.config.MaxMessagesPerTick) // randomly choose how many messages to deliver, up to specified bound
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
					} else {
						if i == 0 {
							// if it's the first tick, drop the message
							messagesToDrop = append(messagesToDrop, next)
						} else {
							// otherwise, keep it for the next
							messages = append(messages, next) // probably not needed, but it is unclear where are the messages stored and read...
						}
					}

				}
			}

			// update report with messages to deliver
			epCtx.Report.AddIntEntry(len(messagesToDeliver), "messages_to_deliver_number", "partitionEnv.handlePartitionCtx")
			for _, m := range messagesToDeliver {
				messageString := fmt.Sprintf("tick: %d | from: %d | to: %d | %s", i, m.From(), m.To(), m.Hash())
				epCtx.Report.AddStringEntry(messageString, "messages_to_deliver", "partitionEnv.handlePartitionCtx")
			}

			// update report with messages to drop
			epCtx.Report.AddIntEntry(len(messagesToDrop), "messages_to_drop_number", "partitionEnv.handlePartitionCtx")
			for _, m := range messagesToDrop {
				messageString := fmt.Sprintf("tick: %d | from: %d | to: %d | %s", i, m.From(), m.To(), m.Hash())
				epCtx.Report.AddStringEntry(messageString, "messages_to_drop", "partitionEnv.handlePartitionCtx")
			}

			if len(messagesToDeliver) > 0 {
				start = time.Now()
				_, err = p.UnderlyingEnv.DeliverMessages(messagesToDeliver, epCtx)
				epCtx.Report.AddTimeEntry(time.Since(start), "deliver_messages_times", "partitionEnv.handlePartitionCtx")
				if err != nil {
					return nil, err
				}
			}
			if len(messagesToDrop) > 0 {
				start = time.Now()
				_, err = p.UnderlyingEnv.DropMessages(messagesToDrop, epCtx)
				epCtx.Report.AddTimeEntry(time.Since(start), "drop_messages_times", "partitionEnv.handlePartitionCtx")
				if err != nil {
					return nil, err
				}
			}
		}

		tickPassedTime := time.Since(tickStartTime).Milliseconds()
		start = time.Now()
		s, err = p.UnderlyingEnv.Tick(epCtx, int(tickPassedTime)) // make the tick pass on the environment
		epCtx.Report.AddTimeEntry(time.Since(start), "tick_time", "partitionEnv.handlePartitionCtx")
		if err != nil {
			return nil, err
		}
	}

	p.messages = make(map[string]Message)
	for k, m := range s.PendingMessages() { // save pending messages for next state
		p.messages[k] = m
	}

	for i := 0; i < p.NumReplicas; i++ {
		id := uint64(i + 1)
		rs := s.GetReplicaState(id)
		color := p.painter.Color(rs)
		if _, ok := nextState.ActiveNodes[id]; !ok {
			color = &InActiveColor{Color: color}
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
	if isSamePartition(nextState.Partition, p.CurPartition.Partition) {
		newRepeatCount = p.CurPartition.RepeatCount + 1
		if newRepeatCount > p.config.StaySameStateUpto {
			newRepeatCount = p.config.StaySameStateUpto
		}
	}
	nextState.RepeatCount = newRepeatCount

	select {
	case <-epCtx.Context.Done():
		return nil, fmt.Errorf("handlePartitionCtx : episode timed out")
	default:
	}
	return nextState, nil
}

func (p *PartitionEnv) handleStartStop(a Action, epCtx *EpisodeContext) (*Partition, error) {
	var start time.Time

	ss, ok := a.(*StopStartAction)
	if !ok {
		return copyPartition(p.CurPartition), fmt.Errorf("handleStartStopCtx : episode timed out")
	}

	// Copy the cur state
	nextState := copyPartition(p.CurPartition)

	// create a new map of active nodes
	newActive := make(map[uint64]bool)
	for i := range p.CurPartition.ActiveNodes {
		newActive[i] = true
	}
	if ss.Action == "Stop" {
		nextState.RemainingCrashes = nextState.RemainingCrashes - 1
		// Stop random node of the color if active
		// build the list of active nodes of that color
		activeNodes := make([]uint64, 0)
		for node, c := range p.CurPartition.ReplicaColors {
			if c.Hash() == ss.Color {
				activeNodes = append(activeNodes, node)
			}
		}
		// if there are active nodes of that color
		if len(activeNodes) > 0 {
			// choose a random node
			nodeI := p.rand.Intn(len(activeNodes))
			epCtx.Report.AddIntEntry(nodeI, "stop_node_index", "partitionEnv.handleStartStopCtx")
			node := activeNodes[nodeI]

			start = time.Now()
			err := p.UnderlyingEnv.Stop(node, epCtx) // stop the node
			epCtx.Report.AddTimeEntry(time.Since(start), "node_stop_time", "partitionEnv.handleStartStopCtx")
			if err != nil {
				return nil, err
			}

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
			epCtx.Report.AddIntEntry(nodeI, "start_node_index", "partitionEnv.handleStartStopCtx")
			node := inactiveNodes[nodeI]

			start = time.Now()
			err := p.UnderlyingEnv.Start(node, epCtx)
			epCtx.Report.AddTimeEntry(time.Since(start), "node_start_time", "partitionEnv.handleStartStopCtx")
			if err != nil {
				return nil, err
			}

			newActive[node] = true
		}
	}
	// Update active nodes in Partition
	nextState.ActiveNodes = make(map[uint64]bool)
	for n := range newActive {
		nextState.ActiveNodes[n] = true
	}

	start = time.Now()
	s, err := p.UnderlyingEnv.Tick(epCtx, 0)
	epCtx.Report.AddTimeEntry(time.Since(start), "tick_time", "partitionEnv.handleStartStopCtx")
	if err != nil {
		return nil, err
	}

	p.messages = make(map[string]Message)
	for k, m := range s.PendingMessages() { // save pending messages for next state
		p.messages[k] = m
	}

	start = time.Now()
	for i := 0; i < p.NumReplicas; i++ {
		id := uint64(i + 1)
		rs := s.GetReplicaState(id)
		_, active := nextState.ActiveNodes[id]
		color := p.painter.Color(rs)
		if !active {
			color = &InActiveColor{Color: color}
		}
		nextState.ReplicaColors[id] = color // abstraction
		nextState.ReplicaStates[id] = rs    // actual replica state
	}
	epCtx.Report.AddTimeEntry(time.Since(start), "get_states_time", "partitionEnv.handleStartStopCtx")

	// Recolor based on active nodes and recompute the partition
	newPartition := make([][]uint64, len(p.CurPartition.Partition))
	inactivePart := make([]uint64, 0)
	newActivePart := make([]uint64, 0)
	for i := range p.CurPartition.Partition {
		newPartition[i] = make([]uint64, 0)
	}
	for n, part := range p.CurPartition.PartitionMap {
		_, newActive := newActive[n]
		_, oldActive := p.CurPartition.ActiveNodes[n]

		if newActive && oldActive {
			newPartition[part] = append(newPartition[part], n)
		} else if newActive && !oldActive {
			newActivePart = append(newActivePart, n)
		} else {
			inactivePart = append(inactivePart, n)
		}
	}
	newPartition = append(newPartition, newActivePart)
	newPartition = append(newPartition, inactivePart)

	tPartition := make([][]uint64, 0)
	for _, part := range newPartition {
		if len(part) == 0 {
			continue
		}
		tPartition = append(tPartition, part)
	}
	newPartition = tPartition
	for i, part := range newPartition {
		for _, node := range part {
			nextState.PartitionMap[node] = i
		}
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
	nextState.PendingRequests = s.PendingRequests()
	nextState.CanDeliverRequest = s.CanDeliverRequest()

	if isSamePartition(nextStatePartition, p.CurPartition.Partition) {
		nextState.RepeatCount = nextState.RepeatCount + 1
		if nextState.RepeatCount > p.config.StaySameStateUpto {
			nextState.RepeatCount = p.config.StaySameStateUpto
		}
	} else {
		nextState.RepeatCount = 0
	}

	select {
	case <-epCtx.Context.Done():
		return nil, fmt.Errorf("handleStartStopCtx : episode timed out")
	default:
	}
	return nextState, nil
}

func (p *PartitionEnv) reset(epCtx *EpisodeContext) error {
	start := time.Now()
	s, err := p.UnderlyingEnv.Reset(epCtx) // takes the state given by the reset function of the underlying environment
	if err != nil {
		return err
	}
	epCtx.Report.AddTimeEntry(time.Since(start), "env_reset_times", "partitionEnv.resetCtx")

	colors := make([]Color, p.NumReplicas)
	curPartition := &Partition{
		ReplicaColors:      make(map[uint64]Color),
		PartitionMap:       make(map[uint64]int),
		ReplicaStates:      make(map[uint64]ReplicaState),
		Partition:          make([][]Color, 1),
		RepeatCount:        0,
		WithCrashes:        p.config.WithCrashes,
		RemainingCrashes:   p.config.CrashLimit,
		MaxInactive:        p.config.MaxInactive,
		ActiveNodes:        make(map[uint64]bool),
		PendingRequests:    s.PendingRequests(),
		CanDeliverRequest:  s.CanDeliverRequest(),
		WithByzantine:      p.config.WithByzantine,
		RemainingByzantine: p.config.MaxByzantine,
		ByzantineNodes:     make(map[uint64]bool),
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

	select {
	case <-epCtx.Context.Done():
		return fmt.Errorf("resetCtx : episode timed out")
	default:
	}
	p.CurPartition = curPartition
	p.messages = make(map[string]Message)
	for k, m := range s.PendingMessages() {
		p.messages[k] = m
	}

	return nil
}

func (p *PartitionEnv) Reset(epCtx *EpisodeContext) (State, error) { // TODO: propagate errors
	err := p.reset(epCtx)
	if err != nil {
		return nil, err
	}
	return copyPartition(p.CurPartition), nil
}

func (p *PartitionEnv) handleByzantine(a Action, epCtx *EpisodeContext) (*Partition, error) {
	byzEnv, ok := p.UnderlyingEnv.(ByzantineEnvironment)
	if !ok {
		return copyPartition(p.CurPartition), fmt.Errorf("handleByzantineCtx : to define")
	}
	ba, ok := a.(*ByzantineAction)
	if !ok {
		return copyPartition(p.CurPartition), fmt.Errorf("handleByzantineCtx : to define")
	}
	nextState := copyPartition(p.CurPartition)
	nonByzantineFilteredNodes := make([]uint64, 0)
	for node, c := range p.CurPartition.ReplicaColors {
		_, byzantine := p.CurPartition.ByzantineNodes[node]
		if !byzantine && c.Hash() == ba.Color {
			nonByzantineFilteredNodes = append(nonByzantineFilteredNodes, node)
		}
	}

	if len(nonByzantineFilteredNodes) == 0 {
		return nextState, fmt.Errorf("handleByzantineCtx : to define")
	}

	select {
	case <-epCtx.Context.Done():
		return nil, fmt.Errorf("resetCtx : episode timed out")
	default:
	}
	toMakeByzantine := nonByzantineFilteredNodes[0]
	byzEnv.BecomeByzantine(toMakeByzantine)

	nextState.RemainingByzantine = nextState.RemainingByzantine - 1
	nextState.ByzantineNodes[toMakeByzantine] = true
	return nextState, nil
}

// UTILIY
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
