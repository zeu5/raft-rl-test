# Predicate Heirarchies

A key feature of the RL approach is to use predicate hierarchies to guide the RL exploration. A predicate hierarchy shapes the rewards given to RL which inturn decides the direction of the exploration.

A predicate hierarchy is defined using the `policies.NewRewardMachine` constructor which accepts a final predicate as a `RewardFuncSingle`. A predicate is essentially a function that accepts a state and returns a boolean indicating if the predicate is true or false.

## Benchmark specific predicates

Each benchmark should defines predicates that accept the concrete `ReplicaState` of the benchmark. A base set of simple predicates should be defined that can be used to build on to higher predicates.
