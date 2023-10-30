package raft

import (
	"bytes"

	"github.com/zeu5/raft-rl-test/types"
	"go.etcd.io/raft/v3"
	pb "go.etcd.io/raft/v3/raftpb"
)

// this is a state of the partition environment --- file: types.partition_env.go
// nextState := &Partition{
// 	ReplicaColors: make(map[uint64]Color),
// 	PartitionMap:  make(map[uint64]int),
// 	ReplicaStates: make(map[uint64]ReplicaState), // this should contain the actual states of the replicas according to the protocol implementation
// 	Partition:     make([][]Color, 0),
// 	RepeatCount:   p.curPartition.RepeatCount,
// }

// what is a ReplicaState for Raft? Where is the Log?
// ReplicaState["state"]: raft.Status
// ReplicaState["log"]:	[]pb.Entry

// checks if the log size of a replica decreases throughout an execution
func ReducedLog() func(*types.Trace) bool {
	return func(t *types.Trace) bool {
		replicasLogs := make(map[uint64][]pb.Entry) // map of processID : Log (list of entries)

		for i := 0; i < t.Len(); i++ { // foreach (state, action, new_state, reward) in the trace
			s, _, _, _ := t.Get(i) // take state s
			pS, ok := s.(*types.Partition)
			if ok {
				for replica_id, elem := range pS.ReplicaStates {
					repState := elem.(map[string]interface{}) // cast into map
					curLog := repState["log"].([]pb.Entry)    // cast "log" into list of pb.Entry

					if _, ok := replicasLogs[replica_id]; !ok { // init empty list if previous replica log is not present
						replicasLogs[replica_id] = make([]pb.Entry, 0)
					}

					if len(curLog) < len(replicasLogs[replica_id]) { // check if log size decreased
						return true // BUG FOUND
					}

					replicasLogs[replica_id] = curLog // update previous state log with the current one for next iteration
				}
			}
		}
		return false
	}
}

// checks if a committed entry of a replica has been changed throughout an execution
func ModifiedLog() func(*types.Trace) bool {
	return func(t *types.Trace) bool {
		replicasLogs := make(map[uint64][]pb.Entry) // map of processID : Log (list of entries)

		for i := 0; i < t.Len(); i++ { // foreach (state, action, new_state, reward) in the trace
			s, _, _, _ := t.Get(i) // take state s
			pS, ok := s.(*types.Partition)
			if ok {
				for replica_id, elem := range pS.ReplicaStates {
					repState := elem.(map[string]interface{}) // cast into map
					curLog := repState["log"].([]pb.Entry)    // cast "log" into list of pb.Entry

					if _, ok := replicasLogs[replica_id]; !ok { // init empty list if previous replica log is not present
						replicasLogs[replica_id] = make([]pb.Entry, 0)
					}

					for j := 0; j < len(replicasLogs[replica_id]); j++ { // for the size of the old log (ignore newly appended entries)
						if !eq_entry(curLog[j], replicasLogs[replica_id][j]) { // check if they are equal
							return true // BUG FOUND
						}
					}

					replicasLogs[replica_id] = curLog // update previous state log with the current one for next iteration
				}
			}
		}
		return false
	}
}

// checks if any replica has an inconsistent log w.r.t. other replicas
func InconsistentLogs() func(*types.Trace) bool {
	return func(t *types.Trace) bool {
		for i := 0; i < t.Len(); i++ { // foreach (state, action, new_state, reward) in the trace
			s, _, _, _ := t.Get(i) // take state s
			pS, ok := s.(*types.Partition)
			if ok {
				// make a list of logs, starting at index 0
				logsList := make([][]pb.Entry, 0, len(pS.ReplicaStates))
				for _, value := range pS.ReplicaStates {
					state := value.(map[string]interface{})
					log := state["log"].([]pb.Entry)
					logsList = append(logsList, log)
				}

				for j1 := 0; j1 < len(logsList); j1++ { // for each replica
					for j2 := j1; j2 < len(logsList); j2++ { // for each other replica
						minSize := min(len(logsList[j1]), len(logsList[j2])) // take the minimum length among the two logs

						for k := 0; k < minSize; k++ { // for each entry
							if !eq_entry(logsList[j1][k], logsList[j2][k]) { // check if they are equal
								return true // BUG FOUND
							}
						}
					}
				}
			}
		}
		return false
	}
}

// Check if there are more than one leader in the same term
func MultipleLeaders() func(*types.Trace) bool {
	return func(t *types.Trace) bool {
		// processStates := make(map[uint64]RaftState)
		// processLogs := make(map[uint64][]pb.Entry) // map of processID : Log (list of entries)
		for i := 0; i < t.Len(); i++ { // foreach (state, action, new_state, reward) in the trace
			s, _, _, _ := t.Get(i)         // take state s
			pS, ok := s.(*types.Partition) // cast into partition
			if ok {
				leaders := make(map[int]int)             // map for leaders number, term : count
				for _, state := range pS.ReplicaStates { // for each replica state
					repState := state.(map[string]interface{})
					curState := repState["state"].(raft.Status)                             // cast into raft.Status
					if curState.BasicStatus.SoftState.RaftState.String() == "StateLeader" { // if the current softState of the replica is "StateLeader"
						curTerm := curState.BasicStatus.HardState.Term // take term
						val := 0
						if v, ok := leaders[int(curTerm)]; !ok { // read leaders count for the term
							val = v
						}
						leaders[int(curTerm)] = val + 1 // increase it by one
						if leaders[int(curTerm)] > 1 {  // if more than one leader => BUG FOUND
							return true
						}
					}
				}
			}
		}
		return false
	}
}

// other functions / auxiliary
func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}

func eq_entry(a, b pb.Entry) bool {
	return bytes.Equal(a.Data, b.Data) && a.Index == b.Index && a.Term == b.Term && a.Type == b.Type
}
