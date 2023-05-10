package lpaxos

type Peer struct {
	ID   uint64
	Addr string
}

type Step int

var (
	StepPrepare Step = 0
	StepAck     Step = 1
	StepPropose Step = 2
	StepPromise Step = 3
)

func (s Step) String() string {
	str := map[int]string{
		0: "Prepare",
		1: "Ack",
		2: "Propose",
		3: "Promise",
	}
	return str[int(s)]
}

type LNodeState struct {
	Last    int
	Phase   int
	Leader  uint64
	Step    Step
	Log     *Log
	Decided int
}

func (l LNodeState) Copy() LNodeState {
	return LNodeState{
		Last:   l.Last,
		Phase:  l.Phase,
		Leader: l.Leader,
		Step:   l.Step,
		Log:    l.Log.Copy(),
	}
}

type LPaxosNode struct {
	ID      uint64
	Config  LPaxosConfig
	Peers   []Peer
	State   LNodeState
	Tracker *Tracker
	ticks   int

	pendingMessages []Message
	// Only used by leader
	// When leader changes forward all pending commands
	// to new leader and clear
	pendingCommands []Entry
}

type LPaxosConfig struct {
	ID      uint64
	Peers   []Peer
	Timeout int
}

func NewLPaxosNode(config LPaxosConfig) *LPaxosNode {
	l := &LPaxosNode{
		ID:     config.ID,
		Config: config,
		Peers:  config.Peers,
		State: LNodeState{
			Last:    -1,
			Phase:   -1,
			Log:     NewLog(),
			Step:    StepPrepare,
			Decided: 0,
		},
		Tracker:         NewTracker(),
		pendingMessages: make([]Message, 0),
		pendingCommands: make([]Entry, 0),
	}
	l.updateToPhase(0)
	return l
}

func (l *LPaxosNode) Status() LNodeState {
	return l.State.Copy()
}

func (l *LPaxosNode) Tick() {
	l.ticks += 1
	if l.State.Leader == l.ID {
		// If leader send heartbeat every (timeout/5) ticks
		if l.ticks >= (l.Config.Timeout / 5) {
			l.broadcast(Message{
				Type:  HeartbeatMessage,
				Phase: l.State.Phase,
			})
			l.ticks = 0
		}
		return
	}
	if l.ticks > l.Config.Timeout {
		l.broadcast(Message{
			Type:    PhaseChangeMessage,
			Phase:   l.State.Phase + 1,
			Last:    l.State.Last,
			Log:     l.State.Log.Entries(),
			LogHash: l.State.Log.Hash(),
		})
	}
}

func (l *LPaxosNode) Step(m Message) {
	if m.Phase < l.State.Phase || !validMessage(m) {
		// Old message or invalid, ignore
		return
	}
	if m.Phase > l.State.Phase {
		// Update to phase
		l.updateToPhase(m.Phase)
	}
	switch m.Type {
	case HeartbeatMessage:
		if l.State.Leader != l.ID && m.From == l.State.Leader {
			// reset ticks when received heartbeat from the leader
			l.ticks = 0
		}
	case PhaseChangeMessage:
		l.Tracker.TrackPhaseChange(Ack{
			Peer:    m.From,
			Last:    m.Last,
			Phase:   m.Phase,
			Log:     m.Log,
			LogHash: m.LogHash,
		})
		n := len(l.Peers)
		if l.Tracker.ValidPhaseChanges(m.Phase) > n/2 {
			if l.getLeader(m.Phase) == l.ID {
				l.updateToPhase(m.Phase)
				l.becomeLeader()
			} else {
				l.broadcast(Message{
					Type:    PhaseChangeMessage,
					Phase:   m.Phase,
					Last:    l.State.Last,
					Log:     l.State.Log.Entries(),
					LogHash: l.State.Log.Hash(),
				})
			}
		}
	case CommandMessage:
		l.pendingCommands = append(l.pendingCommands, m.Log...)
		if l.State.Leader != l.ID {
			l.forwardCommands()
		}
		// If we are waiting for commands to propose then
		// this call will send the propose messages
		if l.State.Step >= StepPropose {
			l.propose()
		}
		return
	case PrepareMessage:
		if l.State.Leader == l.ID {
			l.State.Step = StepAck
		} else {
			l.State.Step = StepPropose
		}
		l.send(Message{
			To:      l.State.Leader,
			Type:    AckMessage,
			Last:    l.State.Last,
			Phase:   l.State.Phase,
			Log:     l.State.Log.Entries(),
			LogHash: l.State.Log.Hash(),
		})
	case AckMessage:
		if l.State.Step < StepAck {
			return
		}
		l.Tracker.TrackAck(Ack{
			Peer:    m.From,
			Phase:   m.Phase,
			Last:    m.Last,
			Log:     m.Log,
			LogHash: m.LogHash,
		})
		n := len(l.Peers)
		if l.Tracker.ValidAcks(m.Phase) > n/2 && l.State.Step == StepAck {
			// propose
			l.propose()
		}
	case ProposeMessage:
		if m.From != l.State.Leader {
			return
		}
		l.State.Log.Replace(m.Log)
		l.State.Last = m.Phase
		l.promise()
	case PromiseMessage:
		l.Tracker.TrackPromise(Promise{
			Peer:    m.From,
			Phase:   m.Phase,
			Log:     m.Log,
			LogHash: m.LogHash,
		})
		n := len(l.Peers)
		if l.Tracker.ValidPromises(l.State.Phase, l.State.Log.Hash()) > n/2 {
			// decide
			l.decide()
		}
	}
}

type Ready struct {
	Messages []Message
}

func (l *LPaxosNode) Ready() Ready {
	messages := make([]Message, len(l.pendingMessages))
	for i, m := range l.pendingMessages {
		messages[i] = m.Copy()
	}
	l.pendingMessages = make([]Message, 0)
	return Ready{
		Messages: messages,
	}
}

func (l *LPaxosNode) HasReady() bool {
	return len(l.pendingMessages) == 0
}

func (l *LPaxosNode) updateToPhase(phase int) {
	if l.State.Phase >= phase {
		// Ignore, already on that phase or beyond
		return
	}
	l.State.Phase = phase
	l.State.Leader = l.getLeader(phase)
	l.State.Step = StepPrepare
	l.Tracker.Reset(phase)

	if len(l.pendingCommands) != 0 && l.State.Leader != l.ID {
		l.forwardCommands()
	} else if l.State.Leader == l.ID {
		// become leader
		l.becomeLeader()
	}
}

func (l *LPaxosNode) becomeLeader() {
	l.broadcast(Message{
		Type:  PrepareMessage,
		Phase: l.State.Phase,
	})
}

func (l *LPaxosNode) getLeader(phase int) uint64 {
	leader := (phase + 1) % len(l.Peers)
	return uint64(leader)
}

func (l *LPaxosNode) forwardCommands() {
	pendingCommands := make([]Entry, len(l.pendingCommands))
	for i, e := range l.pendingCommands {
		pendingCommands[i] = e.Copy()
	}
	l.send(Message{
		To:   l.State.Leader,
		Type: CommandMessage,
		Log:  pendingCommands,
	})
	l.pendingCommands = make([]Entry, 0)
}

func (l *LPaxosNode) send(m Message) {
	m.From = l.ID
	l.pendingMessages = append(l.pendingMessages, m.Copy())
}

func (l *LPaxosNode) broadcast(m Message) {
	m.From = l.ID
	for _, p := range l.Peers {
		m.To = p.ID
		l.pendingMessages = append(l.pendingMessages, m.Copy())
	}
}

func (l *LPaxosNode) propose() {
	if l.State.Leader != l.ID || l.State.Step < StepAck {
		return
	}
	l.State.Step = StepPropose
	var latestLogNode uint64 = 0
	latestAck := -1
	for node, a := range l.Tracker.Acks {
		if a.Last > latestAck {
			latestAck = a.Last
			latestLogNode = node
		}
	}
	if latestAck == -1 {
		// First phase. Pick one at random
		for node := range l.Tracker.Acks {
			latestLogNode = node
			break
		}
	}
	lAck := l.Tracker.Acks[latestLogNode]
	newLog := make([]Entry, len(lAck.Log))
	for i, e := range lAck.Log {
		newLog[i] = e.Copy()
	}
	if len(l.pendingCommands) != 0 {
		newLog = append(newLog, l.pendingCommands...)
		l.pendingCommands = make([]Entry, 0)
	}
	newLogHash := LogHash(newLog)
	if newLogHash == l.State.Log.Hash() {
		// Nothing new to propose
		return
	}
	l.broadcast(Message{
		Type:    ProposeMessage,
		Phase:   l.State.Phase,
		Log:     newLog,
		LogHash: newLogHash,
	})
}

func (l *LPaxosNode) promise() {
	l.State.Step = StepPromise
	l.broadcast(Message{
		Type:    PromiseMessage,
		Phase:   l.State.Phase,
		Log:     l.State.Log.Entries(),
		LogHash: l.State.Log.Hash(),
	})
}

func (l *LPaxosNode) decide() {
	l.State.Decided = l.State.Log.Size()
}
