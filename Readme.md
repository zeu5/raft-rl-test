# RL testing raft and paxos

## Running the experiments

Build the project with `go build .`

To run - `./raft-rl-test <experiment_name>`

Command line parameters:

- `--horizon` to control the horizon of the experiment
- `-e, --episodes` to control the number of episodes

Run `./raft-rl-test <experiment_name> --help` for experiment specific command line options