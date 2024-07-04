## Intro
... is a tool to test distributed consensus protocols implementations. It locally executes an instrumented implementation while simulating the network layer and, at fixed intervals throughout the execution, controlling system events (client requests, network partitions, node failures). It allows to choose among different strategies for choosing the system events. In particular, it includes our Reinforcement Learning based approach which leverages exploration incentiving reward scheme and enables the injection of execution waypoints to guide testing towards specific executions.
The tool is implemented in Go. 

When illustrating through concrete examples, this overview will mostly refer to the specific testing implementation for the RedisRaft benchmark (https://github.com/RedisLabs/redisraft). RedisRaft is the main production-code benchmark in our experimental evaluation and, since it's written in a different language (C) from our tool implementation, it allows to show all the required steps to test a generic implementation.

## HW Dependencies
The tool does not have specific hardware requirements. When run for short and small (limited amount of parallele strategies) experiments, the CPU and Memory consumptions are limited. 
- CPU: executions follows the defined time intervals in the tested implementation execution. This makes running episodes time-consuming, but not CPU intensive. When parallelizing many strategies in a single experiment, the CPU consumption might increase.
- Memory: each episode trace is analyzed directly after its execution. This does not require to keep all the execution traces in memory until the end of the experiment, reducing the amount of memory required to run.
- Disk: The output folder size of an experiment can grow up to a few GBs. This happen when a long experiment is run in combination with enabled output information. The output info (reports, traces) is useful to check whether things are running as expected or not. For example, it is useful for tuning the parameters or check which policy is being learned based on the given waypoints. When running long experiments, it is recommended to disable most of these info features to keep the output folder size limited.

## Getting Started

### Setting up the tool

#### Virtual Machine
guide to import an OVF file with VirtualBox: https://docs.oracle.com/en/virtualization/virtualbox/6.0/user/ovf.html

#### Installation

### Running a short experiment
Go to the tool main folder $INSTALLATION_PATH/raft-rl-test/ (/home/user/app/raft-rl-test/ in the provided VM).

Run:
    ./scripts/run/redis-test.sh

The tool will run for the specified number of episodes (it should take a few minutes at most)

### Processing the output

## Experiments Instructions

### RedisRaft benchmark

## Reusability

### Instrumenting a new implementation

### 