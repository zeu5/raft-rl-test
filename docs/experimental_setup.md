# Experiments

The document lists the experimental setup and results obtained

## Experimental setup

We first describe the environment of Raft and LPaxos

### Raft

The environment `RaftEnvironment` defines `RaftState` which contains:

1. A map of `raft.Status` objects for each node indexed by the node identifier (1..`n` where `n` is the number of nodes).
2. A map of `raftpb.Message` objects of the inflight messages. The map is indexed by the hash of the message.
3. A boolean parameter `WithTimeouts` which defines whether the state allows drop message actions or not

The test harness is defined by the number of requests input for the environment. This is provided as a configuration parameter to `RaftEnvironment`.

Three abstractions over the map of `raft.Status` are defined to be used in the experiments.

1. `ignore-vote` ignores the value of the vote present in the state. This value indicates the node to which the current node has cast its vote to.
2. `ignore-term` ignores the term number in the state
3. `ignore-term-nonleader` ignores the term number of all nodes except the leader

### LPaxos

`LPaxos` implements the algorithm present in - <https://dl.acm.org/doi/pdf/10.1145/3428278> (Figure 1)
The `LPaxosEnv` defines `LPaxosState` similar to Raft and contains

1. A map of `LNodeState` objects
2. A map of `Message` objects
3. A timeout parameter

Two abstractions are defined to be used in the experiments

1. `ignore-last` ignores the last value present in the state
2. `ignore-phase` ignores the phase number of the state of each node

### Benchmarks

The experiment files `raft.go` and `paxos.go` contain a set of RL exploration algorithms to run. The policy defined in the experiments defines the RL algorithm. In particular,

1. The RL algorithm from <https://dl.acm.org/doi/pdf/10.1145/3428298>, labelled `RL-SoftMaxNegPolicy`
2. Pure random exploration, labelled `Random-RandomPolicy`
3. Bonus policy that we designed epsilon-greedy variant, labelled `BonusMaxRL-BonusPolicyGreedy`
4. Bonus policy SoftMax variant, labelled `BonusSoftMaxRL-BonusSoftMax`

One run of the algorithm provides a coverage graph and state visit graph as a json for each policy. The coverage graph plots the number of unique states (map of `raft.Status` in raft and map of `LNodeState` in lpaxos) by the number of iteration

### Variations

Command line parameters allow varying different paremeters of the experiments

1. `-e` specifies the number of iterations (episodes) to run
2. `--horizon` specified the number of steps in each iterations
3. `-r` specifies the number of requests for the test harness
4. `-t` allows the algorithms to have timeouts (message drops) as actions
5. `-a` the abstraction to use for the states

---

## Results

1. With `-t` option, the coverage of `BonusMaxRL-BonusPolicyGreedy` is significantly higher than all other policies in both raft and lpaxos. However,
    - With higher timeout values (lower likelihood of observing higher terms/phases) the curves stabilize for iterations of 10000
    - With lower timeout values the curves stabilize at
2. Without `-t` option, there is no clear winner that provides better coverage
    - TODO: Need to find number of iterations where the curves stabilize
