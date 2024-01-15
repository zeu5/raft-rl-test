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

func NSamePartitionActive(n int) RewardFuncSingle {
	return func(s State) bool {
		p, ok := s.(*Partition)
		if !ok {
			return false
		}
		partitionSizes := make(map[int]int)
		for i := range p.Partition {
			partitionSizes[i] = 0
		}
		for replica, part := range p.PartitionMap {
			if _, active := p.ActiveNodes[replica]; active {
				partitionSizes[part] += 1
				if partitionSizes[part] > n {
					return true
				}
			}
		}
		return false
	}
}
