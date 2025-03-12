# Artifact documentation

## Intro

This repository contains the artifact of the paper "657: Reward Augmentation in Reinforcement Learning for Testing Distributed Systems". Please find below, instructions to set up and run for the two phases of evaluation.

The generic artifact we will later make public provides a framework - WaypointRL - for testing distributed protocol implementations using Reinforcement learning methods and strategies. The framework expects an instrumented implementation as input. The instrumentation will allow the framework to simulate the network and pick the order of messages delivered. Additionally, the framework will control the nodes of the distributed system to introduce failures (stops and restarts). Apart from RL based strategies when testing, the framework provides a generic interface to implement any strategy as an Agent that interacts with the distributed system Environment.

## HW Dependencies

The tool does not have specific hardware requirements. For short experiments, limited resources (CPU, Memory and Disk) suffices. However, for the full evaluation, we recommend running on a system with a minimum of 8CPU cores, 100GB Memory to reproduce results from the paper. The large memory requirement is to accommodate the requirements of RL approaches which store large Q tables in memory.

## Getting Started

### Setting up the tool

We provide a VM that includes prebuilt binaries and dependencies necessary. To perform the evaluation, we recommend using the VM. If it is not possible to run the VM, the code associated with the artifact is available at the following repositories with instructions to build and run.

- [RedisRaft](https://github.com/zeu5/raft-rl-test)
- [Etcd & RSL](https://github.com/zeu5/dist-rl-test)

#### Virtual Machine

Guide to import an OVA file with VirtualBox: https://www.virtualbox.org/manual/UserManual.html#ovf. Requires version 7.0 and above

The VM is available at the following Zenodo link - https://zenodo.org/doi/10.5281/zenodo.12671211

Ideally, there is no login required. However, if necessary, the username for the VM is `user` and password is `pw`.

### Folder structure

Inside the VM the folders are organizes as follows

- `/home/user/app/raft-rl-test/` contains the code for the [RedisRaft](https://github.com/RedisLabs/redisraft) benchmark.
- `/home/user/app/dist-rl-test/` contains the code for the [Etcd](https://github.com/etcd-io/raft) and [RSL](https://github.com/zeu5/go-azure-rsl) benchmarks

## Evaluation instructions

### Running a short experiment

#### For RedisRaft benchmark

The following commands will run for 50 episodes and will take a few minutes to complete. The script will generate the plots and other outputs in the `results/init-test-multi` folder. Specifically, The folder `genericCoverageData` contains the output related to RQ1 (from the paper) and `coverageData` contains the output related to RQ2

```bash
cd /home/user/app/raft-rl-test/
./scripts/run/redis-test-multi.sh
```

#### For Etcd and RSL benchmarks

The following commands will run the experiment for RQ1 of the paper for Etcd benchmark.

```bash
cd /home/user/app/dist-rl-test/
./scripts/run_cov.sh etcd 1000
```

And the following for RQ2

```bash
cd /home/user/app/dist-rl-test/
./scripts/run_hierarchy.sh etcd set1 1000
```

To run the RSL benchmark replace `etcd` with `rsl`

### Full evaluation instructions

#### RedisRaft benchmark

Run the following command for the full evaluation.

```bash
./scripts/run/redis-benchmark.sh $(SET) $(TIMELIMIT) $(ITERATIONS)
```

The values for `SET, TIMELIMIT` and `ITERATIONS` are as follows,

- SET: set7, set8. Most of the experimental evaluation waypoints sequences are contained in one of these two sets.
- TIMELIMIT: short, medium, std, flash. The corresponding timeouts are 30m, 1h, 8h, 5m respectively.
- ITERATIONS: 10. Number of times the experiment is repeated. So with `std` TIMELIMIT, it will run the experiment 10 times for 8h each. Results will be averaged by output processing scripts.

To obtain results similar to the paper, we recommend 5 ITERATIONS with `medium` TIMELIMIT.

#### Etcd and RSL benchmarks

To run full experiment of the Etcd benchmark, run the following commands

For RQ1

```bash
cd /home/user/app/dist-rl-test/
./scripts/run_cov.sh etcd 10000 --num-runs 10
```

For RQ2

```bash
cd /home/user/app/dist-rl-test/
./scripts/run_hierarchy.sh etcd set1 10000 --num-runs 10
```

Replace `etcd` with `rsl` for the RSL benchmark

## Reusability

The framework can be reused to test other implementations of distributed protocols. In what follows, we list the requirements that need to be fulfilled to add a new benchmark. Additionally, the framework allows implementing new strategies to test implementations.

The required steps to test a new implementation are the following:

- instrument the implementation to test
- implement the intercept network functionalities
- map these functionalities in a Reinforcement Learning environment
- define abstractions and predicates
- write the benchmark go file with all the running configurations

When necessary, we relate to the RedisRaft benchmark for concrete examples.

### Instrumenting a new implementation

This step is specific of the implementation to test. In practice, it is sufficient to redirect the outgoing messages of the nodes to the intercept network and expose the nodes states and eventual functionalities.
In other words, the intercept network (implemented in Go in the tool), should receive all the messages sent by the nodes, receive or be able to query the node states, and be able to send a client request or crash/stop a node.

### Implementing the intercept network

The intercept network functionalities should be implemented as shown in the files `raft-rl-test/redisraft/network.go` and `raft-rl-test/redisraft/cluster.go`.
In practice, this code should implement the following functionalities:

- receiving, parsing and sending nodes messages
- starting, stopping and crashing nodes, and restart the whole cluster at each new episode
- sending client requests to the nodes
- querying/receiving and parsing the nodes states

### Defining the RL environment

Once all the functionalities of the intercept network are defined, these should be mapped into a reinforcement learning environment. This can be seen in the file `raft-rl-test/redisraft/env.go`.
In other words, the system should run in episodes consisting in sequences of states and actions. The code of the environment implement actions like:

- restart: recreate the cluster and start a new episode
- step(action): apply the chosen action for the duration of a timestep (ex. deliver messages according to the chosen partition, send a client request to a node, stop a node, ...)
- after each step, it should return the state of the cluster read from the nodes

### Defining abstractions and predicates

Abstractions are defined as colors, they are ways to define which parts of the state is considered while testing. Predicates are boolean functions over states, they are used to define waypoints in the state space.
The user can define many and then use them during testing with different configurations. The code is shown in files `raft-rl-test/redisraft/color.go` and `raft-rl-test/redisraft/predicates.go`.

### Writing the benchmark file

The benchmark file, where the experiment can be designed and all the configurations and parameters can be set, can be written starting from `raft-rl-test/benchmarks/benchmark_template.go`. The file contains minimal parts of the code specific to RedisRaft that should be substituted with the code implemented for the new benchmark.
The analyzers are implemented in the RedisRaft code, but the code is completely generic except from the type of 'colors' they accept. It is sufficient to copy and paste the code and change the type of abstractions they use with the newly defined ones for the benchmark.

### Implementing new policies

In addition to adding a new benchmark, the code allows for implementing new strategies to test. Consider the file `raft-rl-test/types/policies.go` which defines the interface `Policy` and instantiates two concrete policies - `RandomPolicy` and `SoftMaxNegPolicy`. This file serves as a template to guide users in implementing a new policy.