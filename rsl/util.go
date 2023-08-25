package rsl

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func IsProposalUptoDate(decree int, ballot Ballot, compare Proposal) bool {
	return (decree == compare.Decree && ballot.Num > compare.Ballot.Num) || (decree == compare.Decree+1 && ballot.Num == compare.Ballot.Num)
}

func copyMessages(messages map[string]Message) map[string]Message {
	out := make(map[string]Message)
	for k, m := range messages {
		out[k] = m.Copy()
	}
	return out
}

func copyReplicaStates(states map[uint64]LocalState) map[uint64]LocalState {
	out := make(map[uint64]LocalState)
	for id, s := range states {
		out[id] = s.Copy()
	}
	return out
}
