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

type Color int

type Painter interface {
	Color(ReplicaState) Color
}

type ReplicaState interface{}

type Message interface {
	From() uint64
	To() uint64
	Hash() string
}

type PartitionedSystemState interface {
	GetReplicaState(uint64) ReplicaState
	PendingMessages() map[string]Message
}

type PartitionedSystemEnvironment interface {
	Reset() PartitionedSystemState
	Tick() PartitionedSystemState
	DeliverMessage(Message) PartitionedSystemState
}

type Partition struct {
	ReplicaColors map[uint64]Color
	PartitionMap  map[uint64]int
	ReplicaStates map[uint64]ReplicaState
	Partition     [][]Color
}

type CreatePartitionAction struct {
	Partition [][]Color
}

type KeepSamePartitionAction struct{}

func (k *KeepSamePartitionAction) Hash() string {
	return "KeepSamePartition"
}

type PartitionSlice [][]Color

func (p PartitionSlice) Len() int { return len(p) }
func (p PartitionSlice) Less(i, j int) bool {
	if len(p[i]) < len(p[j]) {
		return true
	}
	less := false
	for k := 0; k < len(p[j]); k++ {
		if int(p[i][k]) < int(p[j][k]) {
			less = true
			break
		}
	}
	return less
}
func (p PartitionSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

type ColorSlice []Color

func (c ColorSlice) Len() int { return len(c) }
func (c ColorSlice) Less(i, j int) bool {
	return int(c[i]) < int(c[j])
}
func (c ColorSlice) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c *CreatePartitionAction) Hash() string {
	bs, _ := json.Marshal(c.Partition)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

func (p *Partition) Actions() []Action {
	colorElems := make([]util.Elem, 0)
	for _, c := range p.ReplicaColors {
		colorElems = append(colorElems, util.IntElem(int(c)))
	}
	partitionActions := make([]Action, 0)
	partitionActions = append(partitionActions, &KeepSamePartitionAction{})
	for _, p := range util.EnumeratePartitions(util.MultiSet(colorElems)) {
		partition := make([][]Color, len(p))
		for i, ps := range p {
			partition[i] = make([]Color, len(ps))
			for j, c := range ps {
				partition[i][j] = Color(int(c.(util.IntElem)))
			}
		}
		partitionActions = append(partitionActions, &CreatePartitionAction{Partition: partition})
	}
	return partitionActions
}

func (p *Partition) Hash() string {
	bs, _ := json.Marshal(p.Partition)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

var _ State = &Partition{}
var _ Action = &CreatePartitionAction{}

func copyPartition(p *Partition) *Partition {
	n := &Partition{
		ReplicaColors: make(map[uint64]Color),
		Partition:     make([][]Color, len(p.Partition)),
	}
	for i, s := range p.ReplicaColors {
		n.ReplicaColors[i] = Color(int(s))
	}
	for i, par := range p.Partition {
		n.Partition[i] = make([]Color, len(par))
		for j, col := range par {
			n.Partition[i][j] = Color(int(col))
		}
	}
	return n
}

type PartitionEnv struct {
	NumReplicas            int
	underlyingEnv          PartitionedSystemEnvironment
	painter                Painter
	curPartition           *Partition
	messages               map[string]Message
	ticksBetweenPartitions int
	maxMessagesPerTick     int
	rand                   *rand.Rand
}

var _ Environment = &PartitionEnv{}

type PartitionEnvConfig struct {
	Painter                Painter
	Env                    PartitionedSystemEnvironment
	TicketBetweenPartition int
	NumReplicas            int
	MaxMessagesPerTick     int
}

func NewPartitionEnv(c PartitionEnvConfig) *PartitionEnv {
	p := &PartitionEnv{
		NumReplicas:            c.NumReplicas,
		ticksBetweenPartitions: c.TicketBetweenPartition,
		painter:                c.Painter,
		underlyingEnv:          c.Env,
		messages:               make(map[string]Message),
		curPartition:           nil,
		maxMessagesPerTick:     c.MaxMessagesPerTick,
		rand:                   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	p.reset()
	return p
}

func (p *PartitionEnv) reset() {
	s := p.underlyingEnv.Reset()
	colors := make([]Color, p.NumReplicas)
	curPartition := &Partition{
		ReplicaColors: make(map[uint64]Color),
		PartitionMap:  make(map[uint64]int),
		ReplicaStates: make(map[uint64]ReplicaState),
		Partition:     make([][]Color, 1),
	}
	for i := 0; i < p.NumReplicas; i++ {
		id := uint64(i + 1)
		rs := s.GetReplicaState(id)
		color := p.painter.Color(rs)
		colors[i] = color
		curPartition.ReplicaStates[id] = rs
		curPartition.PartitionMap[id] = 0
		curPartition.ReplicaColors[id] = color
	}
	sort.Sort(ColorSlice(colors))
	curPartition.Partition[0] = colors
	p.curPartition = curPartition
	p.messages = make(map[string]Message)
	for k, m := range s.PendingMessages() {
		p.messages[k] = m
	}
}

func (p *PartitionEnv) Reset() State {
	p.reset()
	return copyPartition(p.curPartition)
}

func (p *PartitionEnv) Step(a Action) State {
	// 1. Change partition
	// 2. Perform ticks, delivering messages in between
	// 3. Update states, partitions and return
	_, keepSame := a.(*KeepSamePartitionAction)

	nextState := &Partition{
		ReplicaColors: make(map[uint64]Color),
		PartitionMap:  make(map[uint64]int),
		ReplicaStates: make(map[uint64]ReplicaState),
		Partition:     make([][]Color, 0),
	}
	newPartition := make([][]uint64, len(p.curPartition.Partition))
	newPartitionMap := make(map[uint64]int)

	if !keepSame {
		ca, _ := a.(*CreatePartitionAction)

		// 1. Change partition
		coloredReplicas := make(map[int][]uint64)
		for r, c := range p.curPartition.ReplicaColors {
			if _, ok := coloredReplicas[int(c)]; !ok {
				coloredReplicas[int(c)] = make([]uint64, 0)
			}
			coloredReplicas[int(c)] = append(coloredReplicas[int(c)], r)
		}
		newPartition = make([][]uint64, len(ca.Partition))
		for i, p := range ca.Partition {
			newPartition[i] = make([]uint64, len(p))
			for j, c := range p {
				nextReplica := coloredReplicas[int(c)][0]
				coloredReplicas[int(c)] = coloredReplicas[int(c)][1:]
				newPartition[i][j] = nextReplica
				newPartitionMap[nextReplica] = i
			}
		}
	} else {
		for i := range p.curPartition.Partition {
			newPartition[i] = make([]uint64, 0)
		}
		for r, i := range p.curPartition.PartitionMap {
			newPartition[i] = append(newPartition[i], r)
			newPartitionMap[r] = i
		}
	}

	// 2. Perform ticks, delivering messages in between
	allMessages := make([]Message, len(p.messages))
	i := 0
	for _, m := range p.messages {
		allMessages[i] = m
		i++
	}
	if len(allMessages) > 0 {
		p.rand.Shuffle(len(allMessages), func(i, j int) {
			allMessages[i], allMessages[j] = allMessages[j], allMessages[i]
		})
	}
	var s PartitionedSystemState
	for i := 0; i < p.ticksBetweenPartitions; i++ {
		mToDeliver := p.rand.Intn(p.maxMessagesPerTick)
		for j := 0; j < mToDeliver; j++ {
			if len(allMessages) > 0 {
				next := allMessages[0]
				allMessages = allMessages[1:]
				fromP := newPartitionMap[next.From()]
				toP := newPartitionMap[next.To()]
				if fromP == toP {
					p.underlyingEnv.DeliverMessage(next)
				}
			}
		}
		s = p.underlyingEnv.Tick()
	}

	// 3. Update states, partitions and return
	p.messages = make(map[string]Message)
	for k, m := range s.PendingMessages() {
		p.messages[k] = m
	}
	for i := 0; i < p.NumReplicas; i++ {
		id := uint64(i + 1)
		rs := s.GetReplicaState(id)
		color := p.painter.Color(rs)
		nextState.ReplicaColors[id] = color
		nextState.ReplicaStates[id] = rs
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
	p.curPartition = copyPartition(nextState)

	return nextState
}
