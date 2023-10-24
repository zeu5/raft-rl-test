# RL testing raft and paxos

## Running the experiments

Build the project with `go build .`

To run - `./raft-rl-test <experiment_name>`

To list all experiments - `./raft-rl-test --help`

For example to run paxos
```
./raft-rl-test paxos-part -e 10000 --horizon 100 --save result_paxos --requests 3 --runs 3
```

Command line parameters:

- `--horizon` to control the horizon of the experiment
- `-e, --episodes` to control the number of episodes

Run `./raft-rl-test <experiment_name> --help` for experiment specific command line options