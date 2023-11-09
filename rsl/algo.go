// Implementing the RSL paxos algorithm in go
// This file contains the main message handlers
// and core protocol logic along with the base
// data types.

// TODO: add reconfiguration
package rsl

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"math/rand"
	"time"
)

// Type wrapper for the base RSL state
type RSLState string

var (
	StateInactive        RSLState = "Inactive"
	StateStablePrimary   RSLState = "StablePrimary"
	StateStableSecondary RSLState = "StableSecondary"
	StateInitializing    RSLState = "Initializing"
	StatePreparing       RSLState = "Preparing"
)

// Command represents a client request
type Command struct {
	Data []byte
}

func (c Command) Copy() Command {
	return Command{
		Data: bytes.Clone(c.Data),
	}
}

func (c Command) Hash() string {
	hash := sha256.Sum256(c.Data)
	return hex.EncodeToString(hash[:])
}

func (c Command) Eq(other Command) bool {
	return bytes.Equal(c.Data, other.Data)
}

// A ballot is defined by the number and the primary for that ballot
type Ballot struct {
	Num  int
	Node uint64
}

func (b Ballot) Copy() Ballot {
	return Ballot{
		Num:  b.Num,
		Node: b.Node,
	}
}

// Proposal to be added to the log
type Proposal struct {
	Ballot  Ballot
	Decree  int
	Command Command
}

func (p Proposal) Copy() Proposal {
	return Proposal{
		Ballot:  p.Ballot.Copy(),
		Decree:  p.Decree,
		Command: p.Command.Copy(),
	}
}

type LocalState struct {
	State               RSLState
	MaxAcceptedProposal Proposal
	MaxPreparedBallot   Ballot
	OldFreshestProposal Proposal
	Decided             int
	PeerConfig          RSLConfig
	Log                 *Log
}

func EmptyLocalState() LocalState {
	return LocalState{
		State: StateInactive,
		Log:   NewLog(),
	}
}

func (l LocalState) Copy() LocalState {
	return LocalState{
		State:               l.State,
		MaxAcceptedProposal: l.MaxAcceptedProposal.Copy(),
		MaxPreparedBallot:   l.MaxPreparedBallot.Copy(),
		OldFreshestProposal: l.OldFreshestProposal.Copy(),
		Decided:             l.Decided,
		PeerConfig:          l.PeerConfig.Copy(),
		Log:                 l.Log.Copy(),
	}
}

type RemoteState struct {
	Node           uint64
	MaxBallot      Ballot
	MaxChosen      int
	LastChosenTime int
}

type Node struct {
	ID uint64
	LocalState

	config *NodeConfig

	commandQ       []Command
	proposalQ      []Proposal
	readyToPropose bool

	// Things to increment on tick
	actionTimeout    int
	noProgressTime   int
	nextElectionTime int
	lastChosenTime   int
	ticks            int

	remoteStates map[uint64]RemoteState
	learningFrom uint64

	pendingMessagesToSend []Message
	rand                  *rand.Rand
}

type NodeConfig struct {
	ID                      uint64
	Peers                   []uint64
	HeartBeatInterval       int
	NoProgressTimeout       int
	BaseElectionDelay       int
	InitializeRetryInterval int
	NewLeaderGracePeriod    int
	VoteRetryInterval       int
	PrepareRetryInterval    int
	MaxCachedLength         int
	ProposalRetryInterval   int
}

func (n NodeConfig) Copy() NodeConfig {
	c := NodeConfig{
		ID:                      n.ID,
		Peers:                   make([]uint64, len(n.Peers)),
		HeartBeatInterval:       n.HeartBeatInterval,
		NoProgressTimeout:       n.NoProgressTimeout,
		BaseElectionDelay:       n.BaseElectionDelay,
		InitializeRetryInterval: n.InitializeRetryInterval,
		NewLeaderGracePeriod:    n.NewLeaderGracePeriod,
		VoteRetryInterval:       n.VoteRetryInterval,
		PrepareRetryInterval:    n.PrepareRetryInterval,
		MaxCachedLength:         n.MaxCachedLength,
		ProposalRetryInterval:   n.ProposalRetryInterval,
	}
	copy(c.Peers, n.Peers)
	return c
}

func NewNode(c *NodeConfig) *Node {
	initialConfig := RSLConfig{Members: make(map[uint64]bool)}
	for _, p := range c.Peers {
		initialConfig.Members[p] = true
	}

	return &Node{
		ID:                    c.ID,
		LocalState:            LocalState{State: StateInactive, PeerConfig: initialConfig, Log: NewLog()},
		config:                c,
		commandQ:              make([]Command, 0),
		proposalQ:             make([]Proposal, 0),
		readyToPropose:        false,
		actionTimeout:         math.MaxInt,
		noProgressTime:        math.MaxInt,
		nextElectionTime:      math.MaxInt,
		lastChosenTime:        0,
		ticks:                 0,
		remoteStates:          make(map[uint64]RemoteState),
		learningFrom:          0,
		pendingMessagesToSend: make([]Message, 0),
		rand:                  rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (n *Node) IsPrimary() bool {
	return n.LocalState.State == StateStablePrimary
}

func (n *Node) Start() {
	n.actionTimeout = math.MaxInt
	n.nextElectionTime = math.MaxInt
	n.lastChosenTime = 0
	n.noProgressTime = n.config.NoProgressTimeout
	n.initialize()
}

func (n *Node) initialize() {
	n.LocalState.State = StateInitializing
	n.remoteStates = make(map[uint64]RemoteState)
	n.broadcast(Message{Type: MessageStatusReq})
	n.actionTimeout = n.ticks + n.config.InitializeRetryInterval
}

type Ready struct {
	Messages []Message
}

func (n *Node) State() LocalState {
	return n.LocalState.Copy()
}

func (n *Node) Tick() {
	// Update the time parameters by one and check for any timeouts
	n.ticks += 1
	if n.ticks > n.noProgressTime {
		// Supposed to quit
		return
	}
	if (n.LocalState.State == StateStablePrimary || n.LocalState.State == StateStableSecondary) && n.MaxPreparedBallot.Node != n.ID && n.ticks > n.nextElectionTime {
		n.initialize()
	}
	if (n.LocalState.State == StateStablePrimary || n.LocalState.State == StateStableSecondary) && n.MaxPreparedBallot.Node == n.ID && n.readyToPropose {
		if n.ticks-n.lastChosenTime > n.config.HeartBeatInterval || len(n.commandQ) > 0 {
			n.sendNextCommand()
		}
	}
	if n.ticks > n.actionTimeout {
		if n.LocalState.State == StateStablePrimary && n.MaxPreparedBallot.Node == n.ID {
			n.sendNextCommand()
		} else if n.LocalState.State == StateInitializing {
			n.initialize()
		} else if n.LocalState.State == StatePreparing {
			n.prepare()
		}
	}
}

func (n *Node) Propose(c Command) {
	n.commandQ = append(n.commandQ, c.Copy())
}

func (n *Node) sendNextCommand() {
	if n.LocalState.State != StateStablePrimary || n.MaxPreparedBallot.Node != n.ID {
		return
	}
	if n.MaxPreparedBallot.Num != n.MaxAcceptedProposal.Ballot.Num {
		return
	}
	n.remoteStates = make(map[uint64]RemoteState)
	if n.readyToPropose && len(n.commandQ) > 0 {
		newProposal := Proposal{
			Ballot:  n.MaxAcceptedProposal.Ballot.Copy(),
			Decree:  n.MaxAcceptedProposal.Decree + 1,
			Command: n.commandQ[0].Copy(),
		}
		n.broadcast(Message{
			Type:    MessageAcceptReq,
			Ballot:  newProposal.Ballot,
			Decree:  newProposal.Decree,
			Command: newProposal.Command,
		})
		n.logProposal(newProposal)
		n.commandQ = n.commandQ[1:]
		n.readyToPropose = false
	} else {
		n.broadcast(Message{
			Type:    MessageAcceptReq,
			Ballot:  n.MaxAcceptedProposal.Ballot.Copy(),
			Decree:  n.MaxAcceptedProposal.Decree,
			Command: n.MaxAcceptedProposal.Command.Copy(),
		})
	}
	n.actionTimeout = n.ticks + n.config.ProposalRetryInterval
}

func (n *Node) HasReady() bool {
	return len(n.pendingMessagesToSend) != 0
}

func (n *Node) Ready() Ready {
	r := Ready{
		Messages: make([]Message, len(n.pendingMessagesToSend)),
	}
	for i, m := range n.pendingMessagesToSend {
		r.Messages[i] = m.Copy()
	}
	n.pendingMessagesToSend = make([]Message, 0)
	return r
}

func (n *Node) Step(m Message) {
	if n.LocalState.State == StateInactive {
		// Discard message if not active
		return
	}

	switch m.Type {
	case MessageAcceptReq:
		n.handleAcceptRequest(m)
	case MessageAcceptResp:
		if !m.Accept {
			n.handleNotAccept(m)
		} else {
			n.handleAcceptResponse(m)
		}
	case MessageFetchReq:
		n.handleFetchRequest(m)
	case MessageFetchResp:
		n.handleFetchResponse(m)
	case MessagePrepareReq:
		n.handlePrepareRequest(m)
	case MessagePrepareResp:
		if !m.Accept {
			n.handleNotAccept(m)
		} else {
			n.handlePrepareResponse(m)
		}
	case MessageStatusReq:
		n.handleStatusRequest(m)
	case MessageStatusResp:
		n.handleStatusResponse(m)
	}
}

func (n *Node) handleStatusRequest(m Message) {
	n.send(Message{
		To:        m.From,
		Type:      MessageStatusResp,
		Ballot:    n.MaxAcceptedProposal.Ballot,
		Decree:    n.MaxAcceptedProposal.Decree,
		Timestamp: n.lastChosenTime,
	})
}

func (n *Node) handleStatusResponse(m Message) {
	if n.LocalState.State != StateInitializing {
		if m.Ballot.Num >= n.MaxAcceptedProposal.Ballot.Num && m.Decree == n.MaxAcceptedProposal.Decree && len(n.proposalQ) == 0 {
			n.learnVotes(m.From)
			return
		}
	} else {
		n.remoteStates[m.From] = RemoteState{
			Node:           m.From,
			MaxBallot:      m.Ballot.Copy(),
			MaxChosen:      m.Decree,
			LastChosenTime: m.Timestamp,
		}
		if n.isMajority(1 + len(n.remoteStates)) {
			if len(n.proposalQ) > 0 {
				minDecreeNeeded := n.proposalQ[0].Decree
				if minDecreeNeeded < n.MaxAcceptedProposal.Decree {
					return
				}
				higherNodes := make([]uint64, 0)
				for n, rs := range n.remoteStates {
					if rs.MaxChosen >= minDecreeNeeded {
						higherNodes = append(higherNodes, n)
					}
				}
				if len(higherNodes) > 0 {
					nodeToLearn := higherNodes[n.rand.Intn(len(higherNodes))]
					n.learnVotes(nodeToLearn)
					return
				}
			}
			// Bug: Need to do what follows only if we see a majority responses
			minDecreeNeeded := n.MaxAcceptedProposal.Decree
			minNeededBallot := n.MaxAcceptedProposal.Ballot

			filteredNodes := make([]uint64, 0)
			for n, rs := range n.remoteStates {
				if rs.MaxChosen == minDecreeNeeded && rs.MaxBallot.Num > minNeededBallot.Num {
					filteredNodes = append(filteredNodes, n)
				} else if rs.MaxChosen > minDecreeNeeded && rs.MaxBallot.Num >= minNeededBallot.Num {
					filteredNodes = append(filteredNodes, n)
				}
			}
			if len(filteredNodes) > 0 {
				nodeToLearn := filteredNodes[n.rand.Intn(len(filteredNodes))]
				n.learnVotes(nodeToLearn)
				return
			}

			if n.nextElectionTime == math.MaxInt {
				maxChosen := 0
				for _, rs := range n.remoteStates {
					if rs.LastChosenTime > maxChosen {
						maxChosen = rs.LastChosenTime
					}
				}
				n.nextElectionTime = min(n.ticks, maxChosen) + n.config.BaseElectionDelay
			}
			if n.ticks > n.nextElectionTime || n.MaxPreparedBallot.Node == n.ID {
				n.prepare()
			} else {
				n.stabilize(false)
			}
		}
	}
}

func (n *Node) isMajority(count int) bool {
	return count >= n.PeerConfig.QuorumSize()
}

func (n *Node) learnVotes(toLearnFrom uint64) {
	n.send(Message{
		To:     toLearnFrom,
		Type:   MessageFetchReq,
		Decree: n.MaxAcceptedProposal.Decree,
		Ballot: n.MaxAcceptedProposal.Ballot.Copy(),
	})

	n.learningFrom = toLearnFrom
}

func (n *Node) Peers() []uint64 {
	peers := make([]uint64, len(n.config.Peers))
	copy(peers, n.config.Peers)
	return peers
}

func (n *Node) addtoExecutionQueue(p Proposal) {
	if IsConfigCommand(p.Command) {
		newConfig, _ := GetRSLConfig(p.Command)
		newMembers := make(map[uint64]bool)
		for k := range newConfig.Members {
			newMembers[k] = true
		}
		n.PeerConfig = RSLConfig{
			InitialDecree: p.Decree,
			Number:        n.PeerConfig.Number + 1,
			Members:       newMembers,
		}
	}

	n.Log.AddDecided(p.Command.Copy())
	n.Decided = n.Log.NumDecided()
}

func (n *Node) logProposal(p Proposal) {
	n.Log.Add(Entry{
		Accepted: true,
		Ballot:   p.Ballot.Copy(),
		Decree:   p.Decree,
		Command:  p.Command,
	})
	n.MaxAcceptedProposal = p
	n.lastChosenTime = n.ticks
	n.noProgressTime = n.ticks + n.config.NoProgressTimeout
	if p.Ballot.Num > n.MaxPreparedBallot.Num {
		n.MaxPreparedBallot = p.Ballot.Copy()
		n.nextElectionTime = max(n.nextElectionTime, n.ticks) + n.config.NewLeaderGracePeriod
	}
}

func (n *Node) stabilize(asPrimary bool) {
	oldState := n.LocalState.State
	if asPrimary {
		n.LocalState.State = StateStablePrimary
	} else {
		n.LocalState.State = StateStableSecondary
	}
	n.remoteStates = make(map[uint64]RemoteState)
	if n.MaxPreparedBallot.Node == n.ID {
		if oldState != StatePreparing {
			// Assertion
			return
		}
		n.actionTimeout = n.ticks + n.config.VoteRetryInterval
	} else {
		n.actionTimeout = math.MaxInt
	}
}

func (n *Node) prepare() {
	if n.LocalState.State == StateInitializing {
		n.LocalState.State = StatePreparing
		n.MaxPreparedBallot = Ballot{
			Num:  n.MaxPreparedBallot.Num + 1,
			Node: n.ID,
		}
		n.Log.Add(Entry{
			Ballot: n.MaxPreparedBallot.Copy(),
		})
		n.nextElectionTime = n.ticks + n.config.NewLeaderGracePeriod
	} else if n.LocalState.State != StatePreparing {
		// Assertion
		return
	}
	n.OldFreshestProposal = n.MaxAcceptedProposal.Copy()
	n.remoteStates = make(map[uint64]RemoteState)
	// Bug: Need to send ballot here?
	n.broadcast(Message{
		Type:   MessagePrepareReq,
		Decree: n.MaxAcceptedProposal.Decree,
		Ballot: n.MaxPreparedBallot.Copy(),
	})
	n.actionTimeout = n.ticks + n.config.PrepareRetryInterval
}

func (n *Node) handlePrepareRequest(m Message) {
	if m.Ballot.Num < n.MaxPreparedBallot.Num || m.Decree < n.MaxAcceptedProposal.Decree {
		n.send(Message{
			To:     m.From,
			Type:   MessagePrepareResp,
			Accept: false,
			Ballot: n.MaxPreparedBallot.Copy(),
			Decree: n.MaxAcceptedProposal.Decree,
		})
	} else if n.LocalState.State != StateInitializing {
		if m.Ballot.Num > n.MaxPreparedBallot.Num {
			n.MaxPreparedBallot = m.Ballot.Copy()
			n.Log.Add(Entry{
				Accepted: false,
				Ballot:   m.Ballot.Copy(),
			})
			n.nextElectionTime = max(n.ticks, n.nextElectionTime) + n.config.NewLeaderGracePeriod
			n.send(Message{
				To:             m.From,
				Type:           MessagePrepareResp,
				Accept:         true,
				Ballot:         m.Ballot.Copy(),
				ProposalBallot: n.MaxAcceptedProposal.Ballot.Copy(),
				Decree:         n.MaxAcceptedProposal.Decree,
				Command:        n.MaxAcceptedProposal.Command.Copy(),
			})
		}
		if m.Decree > n.MaxAcceptedProposal.Decree {
			n.initialize()
		} else {
			n.stabilize(false)
		}
	}
}

func (n *Node) handlePrepareResponse(m Message) {
	if n.LocalState.State == StatePreparing && n.MaxPreparedBallot.Num == m.Ballot.Num && n.MaxAcceptedProposal.Decree == m.Decree {
		if m.Ballot.Node != n.ID {
			// Assertion
			return
		}
		if m.Decree == n.OldFreshestProposal.Decree && m.ProposalBallot.Num > n.OldFreshestProposal.Ballot.Num {
			n.OldFreshestProposal = Proposal{
				Ballot:  m.ProposalBallot.Copy(),
				Decree:  m.Decree,
				Command: m.Command.Copy(),
			}
		}
		n.remoteStates[m.From] = RemoteState{
			Node:           m.From,
			MaxBallot:      m.ProposalBallot.Copy(),
			MaxChosen:      m.Decree,
			LastChosenTime: 0,
		}
		if n.isMajority(len(n.remoteStates) + 1) {
			n.OldFreshestProposal.Ballot = n.MaxPreparedBallot.Copy()
			n.broadcast(Message{
				Type:    MessageAcceptReq,
				Ballot:  n.OldFreshestProposal.Ballot.Copy(),
				Decree:  n.OldFreshestProposal.Decree,
				Command: n.OldFreshestProposal.Command.Copy(),
			})
			n.logProposal(n.OldFreshestProposal.Copy())
			n.readyToPropose = false
			n.stabilize(true)
		}
	}
}

func (n *Node) handleAcceptRequest(m Message) {
	if m.Ballot.Node != m.From || m.From == n.ID {
		// Assertion
		// Crucially, the Accept message sent to self is ignored
		return
	}
	if n.MaxPreparedBallot.Num < n.MaxAcceptedProposal.Ballot.Num {
		// Assertion
		return
	}
	if n.LocalState.State != StateInitializing {
		if m.Decree < n.MaxAcceptedProposal.Decree || m.Ballot.Num < n.MaxPreparedBallot.Num {
			n.send(Message{
				To:     m.From,
				Type:   MessageAcceptResp,
				Accept: false,
				Ballot: n.MaxPreparedBallot.Copy(),
				Decree: n.MaxAcceptedProposal.Decree,
			})
			return
		} else if m.Decree == n.MaxAcceptedProposal.Decree && m.Ballot.Num == n.MaxAcceptedProposal.Ballot.Num {
			if !m.Command.Eq(n.MaxAcceptedProposal.Command) {
				// Assertion
				return
			}
		} else if (m.Decree == n.MaxAcceptedProposal.Decree && m.Ballot.Num >= n.MaxAcceptedProposal.Ballot.Num) || (m.Decree == n.MaxAcceptedProposal.Decree+1 && m.Ballot.Num == n.MaxAcceptedProposal.Ballot.Num) {
			if m.Decree == n.MaxAcceptedProposal.Decree+1 {
				n.addtoExecutionQueue(n.MaxAcceptedProposal.Copy())
				n.nextElectionTime = n.ticks + n.config.BaseElectionDelay
			}
			n.logProposal(Proposal{
				Ballot:  m.Ballot.Copy(),
				Decree:  m.Decree,
				Command: m.Command.Copy(),
			})
			n.send(Message{
				To:     m.From,
				Type:   MessageAcceptResp,
				Accept: true,
				Ballot: m.Ballot.Copy(),
				Decree: m.Decree,
			})
			n.stabilize(false)
		} else {
			n.initialize()
		}
	} else {
		if len(n.proposalQ) > 0 {
			lastP := n.proposalQ[len(n.proposalQ)-1]
			if m.Decree < lastP.Decree || m.Ballot.Num < lastP.Ballot.Num {
				if m.Decree < n.MaxAcceptedProposal.Decree || m.Ballot.Num < n.MaxAcceptedProposal.Ballot.Num {
					n.send(Message{
						To:     m.From,
						Type:   MessageAcceptResp,
						Accept: false,
						Ballot: n.MaxPreparedBallot.Copy(),
						Decree: n.MaxAcceptedProposal.Decree,
					})
				}
				return
			}
			if m.Decree == lastP.Decree && m.Ballot.Num == lastP.Ballot.Num {
				return
			}
			if len(n.proposalQ) == n.config.MaxCachedLength {
				n.proposalQ = make([]Proposal, 0)
			}
			if (m.Decree == lastP.Decree && m.Ballot.Num > lastP.Ballot.Num) || (m.Decree == lastP.Decree+1 && m.Ballot.Num == lastP.Ballot.Num) {
				n.proposalQ = append(n.proposalQ, Proposal{
					Ballot:  m.Ballot.Copy(),
					Decree:  m.Decree,
					Command: m.Command.Copy(),
				})
			} else {
				n.proposalQ = make([]Proposal, 0)
				n.proposalQ = append(n.proposalQ, Proposal{
					Ballot:  m.Ballot.Copy(),
					Decree:  m.Decree,
					Command: m.Command.Copy(),
				})
			}
		} else {
			n.proposalQ = append(n.proposalQ, Proposal{
				Ballot:  m.Ballot.Copy(),
				Decree:  m.Decree,
				Command: m.Command.Copy(),
			})
		}
	}
}

func (n *Node) handleNotAccept(m Message) {
	if m.Decree < n.MaxAcceptedProposal.Decree || m.Ballot.Num < n.MaxAcceptedProposal.Ballot.Num {
		return
	}
	if n.LocalState.State != StateInitializing {
		if m.Decree > n.MaxAcceptedProposal.Decree {
			n.initialize()
		} else if m.Ballot.Num > n.MaxPreparedBallot.Num {
			n.MaxPreparedBallot = m.Ballot.Copy()
			n.Log.Add(Entry{
				Accepted: false,
				Ballot:   m.Ballot.Copy(),
			})
			n.nextElectionTime = max(n.nextElectionTime, n.ticks) + n.config.NewLeaderGracePeriod
			n.stabilize(false)
		}
	}
}

func (n *Node) handleAcceptResponse(m Message) {
	if (n.LocalState.State == StateStablePrimary || n.LocalState.State == StateStableSecondary) &&
		n.MaxPreparedBallot.Num == m.Ballot.Num &&
		n.MaxAcceptedProposal.Decree == m.Decree &&
		n.MaxPreparedBallot.Node == n.ID &&
		!n.readyToPropose {
		if n.MaxPreparedBallot.Num != n.MaxAcceptedProposal.Ballot.Num {
			// Assertion
			return
		}
		n.remoteStates[m.From] = RemoteState{
			Node:           m.From,
			MaxBallot:      m.Ballot.Copy(),
			MaxChosen:      m.Decree,
			LastChosenTime: 0,
		}
		if n.isMajority(len(n.remoteStates) + 1) {
			n.addtoExecutionQueue(n.MaxAcceptedProposal)
			n.readyToPropose = true
			n.actionTimeout = math.MaxInt
		}
	}
}

func (n *Node) handleFetchRequest(m Message) {
	validProposals := make([]Proposal, 0)
	for _, e := range n.Log.entries {
		if e.Accepted && e.Decree >= m.Decree {
			validProposals = append(validProposals, Proposal{
				Decree:  e.Decree,
				Ballot:  e.Ballot.Copy(),
				Command: e.Command.Copy(),
			})
		}
	}

	n.send(Message{
		To:        m.From,
		Type:      MessageFetchResp,
		Proposals: validProposals,
	})
}

func (n *Node) handleFetchResponse(m Message) {
	if n.learningFrom == 0 || n.learningFrom != m.From {
		return
	}
	minNeeded := Proposal{}
	haveMinNeeded := false
	if len(n.proposalQ) > 0 {
		minNeeded = n.proposalQ[0]
		haveMinNeeded = true
	}
	defer func(n *Node) {
		n.learningFrom = 0
	}(n)
	for _, p := range m.Proposals {
		if p.Decree <= n.MaxAcceptedProposal.Decree || p.Ballot.Num <= n.MaxAcceptedProposal.Ballot.Num {
			return
		}
		if p.Decree == n.MaxAcceptedProposal.Decree+1 && p.Ballot.Num > n.MaxAcceptedProposal.Ballot.Num {
			// Add to execution queue
			n.addtoExecutionQueue(n.MaxAcceptedProposal)
			n.nextElectionTime = n.ticks + n.config.BaseElectionDelay
			n.logProposal(Proposal{
				Ballot:  n.MaxAcceptedProposal.Ballot.Copy(),
				Decree:  n.MaxAcceptedProposal.Decree + 1,
				Command: p.Command.Copy(),
			})
		}
		if !((p.Decree == n.MaxAcceptedProposal.Decree && p.Ballot.Num >= n.MaxAcceptedProposal.Ballot.Num) || (p.Decree == n.MaxAcceptedProposal.Decree+1 && p.Ballot.Num == n.MaxAcceptedProposal.Ballot.Num)) {
			// Assertion
			return
		}
		if p.Decree == n.MaxAcceptedProposal.Decree+1 {
			n.addtoExecutionQueue(n.MaxAcceptedProposal)
			n.nextElectionTime = n.ticks + n.config.BaseElectionDelay
		}
		n.logProposal(p.Copy())

		if !haveMinNeeded || minNeeded.Decree <= m.Decree {
			for _, p := range n.proposalQ {
				if p.Decree == n.MaxAcceptedProposal.Decree+1 {
					n.addtoExecutionQueue(n.MaxAcceptedProposal)
					n.nextElectionTime = n.ticks + n.config.BaseElectionDelay
				}
				n.logProposal(p)
			}
			n.stabilize(false)
			return
		}
	}
	n.initialize()
}
