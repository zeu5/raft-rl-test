#!/bin/bash

# bash redis_run.sh <sequence_set> <num_episodes> <episode_horizon> <output_path>

# ARGS:
#   1 hierarchy set name - any string to run with no hierarchies
#   2 number of episodes
#   3 episode horizon
#   4 output path

parent_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )
cd "$parent_path"

# delete and create output folder
rm -rf ${4}
mkdir -m 777 ${4}

cd ./../../ # move to main repo folder
go build . # build

# print eventual existing running processes
pgrep redis-server
pgrep raft-rl-test
# kill them
pkill -9 redis-server
pkill -9 raft-rl-test
# wait for few seconds
sleep 3

# run the tool with the specified parameters
./raft-rl-test redisraft-rm $1 -e $2 --horizon $3 --save ${4} # 2>&1 | tee ${4}/outtext.txt
