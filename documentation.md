## Intro
This is a tool to test distributed consensus protocols implementations. It locally executes an instrumented implementation under test while simulating the network layer and, at fixed intervals throughout the execution, controlling system events (client requests, network partitions, node failures). It allows to choose among different strategies for picking which system events will happen throughout the execution. In particular, it includes our Reinforcement Learning based approach which leverages exploration incentiving reward scheme and enables the injection of execution waypoints to guide testing towards specific executions.
The tool is implemented in Go.

When illustrating through concrete examples, this overview will mostly refer to the specific testing implementation for the RedisRaft benchmark (https://github.com/RedisLabs/redisraft). RedisRaft is the main production-code benchmark in our experimental evaluation and, since it's written in a different language (C) from our tool implementation, it allows to show all the required steps to test a generic implementation.

## HW Dependencies
The tool does not have specific hardware requirements. When run for short and small (limited amount of parallele strategies) experiments, the CPU and Memory consumptions are limited. 
- CPU: executions follows the defined time intervals in the tested implementation execution. This makes running episodes time-consuming, but not CPU intensive. When parallelizing many strategies in a single experiment, the CPU consumption might increase.
- Memory: each episode trace is analyzed directly after its execution. This does not require to keep all the execution traces in memory until the end of the experiment, reducing the amount of memory required to run.
- Disk: The output folder size of an experiment can grow up to a few GBs. This happen when a long experiment is run in combination with enabled output information. The output info (reports, traces) is useful to check whether things are running as expected or not. For example, it is useful for tuning the parameters or check which policy is being learned based on the given waypoints. When running long experiments, it is recommended to disable most of these info features to keep the output folder size limited.

## Getting Started

### Setting up the tool
The tool can be used by downloading and building the code directly or by using the provided VMs. The main requirement to run the tool is having Go installed. In addition, the implementation to be tested should also be built and run.

#### Virtual Machine
guide to import an OVA file with VirtualBox: https://docs.oracle.com/en/virtualization/virtualbox/6.0/user/ovf.html

The provided VM image already contains all the dependencies to run the benchmarks which are used in the paper.

#### Installation
To build the tool on another machine, the only requirement is installing Go (https://go.dev/doc/install). Python3 and the library matplotlib are required to run the scripts to process the output and make the plots.

To run the redisraft benchmark, the instrumented code should be built on the machine. In case of linux systems, the pre-built file already contained in the tool might work.
*WE DON'T HAVE DOCUMENTATION ON HOW TO BUILD IT, USING THE PROVIDED VM IS PREFERRED*

### Running a short experiment
Go to the tool main folder $INSTALLATION_PATH/raft-rl-test/ (/home/user/app/raft-rl-test/ in the provided VM).

Run:
    
    ./scripts/run/redis-test.sh

The tool will run for the specified number of episodes (it should take a few minutes at most) and then process the output folder generating the plots inside the coverage folders.

Run:
    
    ./scripts/run/redis-test-multi.sh

For a short test repeated 3 times, with plots and final json files having averaged results. The results can be in found in ./results/init-test or ./results/init-test-multi. Inside the folders genericCoverageData and coverageData.
Inside of each of the subfolders there will be another /output folder containing the coverage plots relative to the final waypoint predicate.

### Processing the output
The output is processed by a set of python scripts. When running the provided bash scripts for init-test and running the benchmarks, these scripts are already called upon experiment termination.

The output folder of each sequential iteration of one experiment (which can contain several strategies run in parallel) should be contained in a single outer folder. 
ex. redis-set7
    |_exp1
    |_exp2
    |_exp3

- ./scripts/graphs/coverage/createDataFolders.py redis-set7 : it will create coverage output folders, containing one .json file for each of the exp iterations.
- ./scripts/graphs/coverage/plotAll.py redis-set7 : creates an output folder inside each of the final subfolder of the coverage output (there is one for each waypoint sequence, representing the state coverage with respect for that final predicate)
    each of the output folder will contain averaged coverage plots (with and without standard deviation)
- ./scripts/graphs/coverage/analyzeRmData.py redis-set7 : will produce some aggregated json files with more info about waypoints sequences

## Experiments Instructions

### RedisRaft benchmark

Run:
    
    ./scripts/run/redis-benchmark.sh $(SET) $(TIMELIMIT) $(ITERATIONS)

- SET: set7, set8\
Most of the experimental evaluation waypoints sequences are contained in one of these two sets.

- TIMELIMIT: short, medium, std, flash\
Respectively 30m, 1h, 8h, 5m. If not specified, reads the value specified in the benchmark file benchmarks/redisraft_rm.go customizable by changing the variable comparisonTimeBudget.

- ITERATIONS: integer value\
Number of times the experiment is executed sequentially. Results will be averaged by output processing scripts.

## Reusability
The required steps to test a new implementation are the following:
- instrument the implementation to test
- implement the intercept network functionalities
- map these functionalities in a Reinforcement Learning environment
- define abstractions and predicates
- write the benchmark go file with all the running configurations

### Instrumenting a new implementation
This step is specific of the implementation to test. In practice, it is sufficient to redirect the outgoing messages of the nodes to the intercept network and expose the nodes states and eventual functionalities.
In other words, the intercept network (implemented in Go in the tool), should receive all the messages sent by the nodes, receive or be able to query the node states, and be able to send a client request or crash/stop a node.

### Implementing the intercept network
The intercept network functionalities should be implemented as shown in the files **/redisraft/network.go** and **/redisraft/cluster.go**. 
In practice, this code should implement the following functionalities:
- receiving, parsing and sending nodes messages
- starting, stopping and crashing nodes, and restart the whole cluster at each new episode
- sending client requests to the nodes
- querying/receiving and parsing the nodes states

### Defining the RL environment
Once all the functionalities of the intercept network are defined, these should be mapped into a reinforcement learning environment. This can be seen in the file **/redisraft/env.go**.
In other words, the system should run in episodes consisting in sequences of states and actions. The code of the environment implement actions like:
- restart: recreate the cluster and start a new episode
- step(action): apply the chosen action for the duration of a timestep (ex. deliver messages according to the chosen partition, send a client request to a node, stop a node, ...)
- after each step, it should return the state of the cluster read from the nodes

### Defining abstractions and predicates
Abstractions are defined as colors, they are ways to define which parts of the state is considered while testing. Predicates are boolean functions over states, they are used to define waypoints in the state space.
The user can define many and then use them during testing with different configurations. The code is shown in files **/redisraft/color.go** and **/redisraft/predicates.go**.

### Writing the benchmark file
The benchmark file, where the experiment can be designed and all the configurations and parameters can be set, can be written starting from **/benchmarks/benchmark_template.go**. The file contains minimal parts of the specific code for redisraft that should be substituted with the code implemented for the new benchmark.
The analyzers are implemented in the redisraft code, but the code is completely generic except from the type of 'colors' they accept. It is sufficient to copypaste the code and change the type of abstractions they use with the newly defined ones for the benchmark. [future plan to polish the code and properly modularize it]

### 