# Benchmark setup

We record the steps to undertake before adding a new benchmark to test using RL. Specifically, the `Environment` interface that needs to be implemented and the instrumentation needed to implement the interface.

Furthermore, for each of the benchmarks we have, we record the specific parameters that fine tune the running environment and impact the performance of the learning.

## Environment interface

The general environment that RL expects is as follows

``` go
type Environment interface {
    // Reset called at the end of each episode
    Reset() State
    // Take the corresponding action and return the next state
    Step(Action) State
}

// State of the system that RL policies observe
type State interface {
    // Indexed by the Hash
    // Should be deterministic
    Hash() string
    // Actions possible from the state
    Actions() []Action
}

// And Action that RL policy can take
type Action interface {
    // Index of the action
    // Should be deterministic
    Hash() string
}
```

However for distributed systems, we define an abstract implementation of the `Environment` interface - `PartitionEnvironment`.

## Partition environment

The partition environment maintains a state where the nodes of the distributed system are assigned specific partitions. Additionally, the Partition environment also maintains the complete concrete state of each node. However, RL only observes an abstraction for each node which we call as `Color`. Specifically, the `PartitionEnvironment.Hash` function returns a hash of the current partition configuration where each node is assigned a specific `Color` and the configuration is maintained as `[][]Color`.

Note: we will use node/replica interchangeably.

The actions of the `PartitionEnvironment` specific to the partitions are as follows.

- `CreatePartition`: reconfigures the partition configuration based on the colors.
- `KeepSamePartition`: retains the current partition configuration.

To enable implementing the semantics of the above actions, `PartitionEnvironment` relies on an underlying concrete implementation of the `PartitionedSystemEnvironment` interface (described below)

``` go
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
```

When RL policy picks one of `CreatePartition` or `KeepSamePartition`, the following steps are executed:

1. A new partition configuration is computed based on the parameter in the action. (no change in the case of `KeepSamePartition`)
2. Based on a configuration `TicksPerPartition`, we invoke `PartitionedSystemEnvironment.Tick` as many times as specified.
3. Before each tick, we collect the set of pending messages from the current state `PartitionedSystemState.PendingMessages`. Divide the messages into (1) those that need to be delivered and (2) those that need to be dropped based on the parittion configuration. We invoke `PartitionedSystemEnvironment.DeliverMessage` for (1) and `PartitionedSystemEnvironment.DropMessage` for (2)
4. We collect the `ReplicaState` of each replica and their respective `Color`.
5. Finally, if the resulting partition configuration of colored nodes is the same as the previous state (despite delivering messages), we increment a state variable `RepeatCount` by one with an upper bound based on the configuration `StaySameStateUpto`.

The color is assigned based on a configured `Painter` that accepts the `ReplicaState` and returns `Color`.

``` go
type Painter interface {
    Color(ReplicaState) Color
}
```

`PartitionedEnvironment` provides RL with the following additional actions:

- `StartStop`: Where a node (identified by the color) can be started or stopped.
- `SendRequest`: Deliver a request to the `PartitionedSystemEnvironment`.

For `StartStop`, the respect `PartitionedSystemEnvironment.Start` or `PartitionedSystemEnvironment.Stop` is called and a counter is maintained for the number of `Stop` actions invoked so far. Similarly, `PartitionedSystemEnvironment.ReceiveRequest` for the `SendRequest` action. However, the `SendRequest` action is enabled only if there are `PendingRequests` and `PartitionedSystemState.CanDeliverRequest` returns true.

To summarize the configuration for a `PartitionEnvironment` is as follows

- A concrete implementation of `PartitionedSystemEnvironment`
- A `Painter` to color the nodes
- `TicksBetweenPartition` parameter
- `NumReplicas` is the number of nodes in the underlying environment to query for a state. The replicas have a default identifier of `[1..NumReplicas]`
- `MaxMessagesPerTick` that decides how many messages between partitions to be scheduled for delivery/dropped.
- `StaySameStateUpto` to decide how many repeated configurations constitute as a new state.
- `WithCrashes` enables the `StartStop` action
- `MaxInactive` upper bounds how many nodes can be inactive (stopped) in a given state
- `CrashLimit` limits the number of `StartStop` actions where the action is to `Stop` can occur in a given episode.

## Benchmark environment

In order to incorporate a benchmark to use the different RL approaches, the framework requires that the benchmark implements the `PartitionedSystemEnvironment` interface. Specifically,

- Implement a mechanism to capture messages to enable `DeliverMessage` and `DropMessage` function
- Ability read and parse the state of each replica/node to implement `GetReplicaState` function including defining the concrete `ReplicaState` structure.
- Define a concrete `Request` structure and instrument the ability to deliver client requests and define the predicate to enable the `SendRequest` action. Specifically, the `CanDeliverRequest` function.
- Control the running of the underlying nodes to implement the `Start` and `Stop` functions.
- Define the granularity of a step that RL can take by defining the `Tick` function.
- Define a painter to read the concrete `ReplicaState` and return a `Color`. Ideally, the painter should be composable to play with different state abstractions.
- Finally, the ability to reset the complete state and start the nodes from scratch. It is important to make sure that the reset is clean (no residual state).

Most importantly, the transitions to `PartitionedSystemState` returned should preserve the markov property (does not depend on the path of the execution that leads to the state).

To enable the above functionalities, a manual instrumentation of the code running in each process is required.

## Specific Benchmark details

### CometBFT (Tendermint)

The first step is to instrument the code in order to capture the messages transmitted. This required implementing a new `Transport` interface in the Tendermint codebase.

The `CometEnv` (cbft/env.go) implements (1) a mock network that captures the intercepted messages transmitted by the instrumented code and (2) a `Cluster` structure that wraps the processes running the nodes. The configuration for creating a new cluster environment is as follows.

``` go
type CometClusterConfig struct {
    NumNodes            int
    CometBinaryPath     string
    InterceptListenPort int
    BaseRPCPort         int
    BaseWorkingDir      string
    NumRequests         int

    TimeoutPropose   int
    TimeoutPrevote   int
    TimeoutPrecommit int
    TimeoutCommit    int
}
```

The details of the configuration is:

- `CometBinaryPath` - path to the binary of the instrumented codebase. Invoking the binary should start one single process
- `InterceptListenPort` - address of the mock network to listen for intercepted messages
- `BaseRPCPort` a configuration for the process RPC port to query for the state and to deliver client requests
- `BaseWorkingDir` - directory to prepare the configuration for each process and to store all the information related to each replica
- `NumRequests` configures how many `SendRequest` actions can be enabled in an episode.
- `Timeout*` the concrete timeout values that the replicas are configured with. The defaults are `Propose = 200ms, Prevote = 100ms, Precommit - 100ms, Commit = 100ms`

Concrete values used:

```
TicketBetweenPartition: 3,
MaxMessagesPerTick:     20,
StaySameStateUpto:      2,
NumReplicas:            4,
WithCrashes:            true,
CrashLimit:             10,
MaxInactive:            2,
```

the colors used:

- `height/round/step`, `proposal`, `number of round votes`, `proposer`
- `NumRequests` is 20.
