package types

func RemainingRequests(min int) RewardFuncSingle {
	return func(s State) bool {
		p, ok := s.(*Partition)
		if !ok {
			return false
		}
		return len(p.PendingRequests) > min
	}
}
