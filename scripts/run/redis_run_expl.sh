#!/bin/bash

# bash redis_run.sh <num_episodes> <episode_horizon> <result_path>

# ARGS:
#   1 number of episodes
#   2 episode horizon
#   3 Result path

parent_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )
cd "$parent_path"

results_path="../../results_redis"

if [ -d $results_path ]; then
    rm -rf $results_path
fi
mkdir -m 777 $results_path
# mkdir ./results_redis_${4}_${5}

# exec > >( tee ~/../../local/aborgare/results_redis_${4}_${5}/outtext.txt) 2>&1

cd ./../../
go build .

pgrep redis-server
pgrep raft-rl-test

pkill -9 redis-server
pkill -9 raft-rl-test

sleep 3

./raft-rl-test redisraft-rm expl-rl -e $1 --horizon $2 --save $results_path 2>&1 | tee $results_path/outtext.txt

# mkdir ./results_redis_${4}_${5}
# ./raft-rl-test redisraft-rm $1 -e $2 --horizon $3 --save ./results_redis_${4}_${5} 2>&1 | tee ./results_redis_${4}_${5}/outtext.txt
