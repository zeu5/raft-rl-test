#!/bin/bash

# bash redis_run.sh <hier_set> <num_episodes> <episode_horizon> <machine name> <exp_number>

# ARGS:
#   1 hierarchy set name - any string to run with no hierarchies
#   2 number of episodes
#   3 episode horizon
#   4 machine name - ex. p19
#   5 experiment number - ex. 02

parent_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )
cd "$parent_path"

# rm -rf /RSE/RLDS/work/aborgare/data/results_redis_${4}_${5}
# mkdir -m 777 /RSE/RLDS/work/aborgare/data/results_redis_${4}_${5}
# mkdir ./results_redis_${4}_${5}

if [ ! -d "/local/aborgare" ]; then
  mkdir -m 777 /local/aborgare
fi

rm -rf /local/aborgare/results_redis_${4}_${5}
mkdir -m 777 /local/aborgare/results_redis_${4}_${5}

# exec > >( tee ~/../../local/aborgare/results_redis_${4}_${5}/outtext.txt) 2>&1

cd ./../../
go build .

cd ./scripts/run/

# output and kill open processes
pgrep redis-server
pgrep raft-rl-test

pkill -9 redis-server
pkill -9 raft-rl-test

sleep 3

# ./../../raft-rl-test redisraft-rm $1 -e $2 --horizon $3 --cpuprofile cpu.prof --memprofile mem.prof --save /RSE/RLDS/work/aborgare/data/results_redis_${4}_${5} 2>&1 | tee /RSE/RLDS/work/aborgare/data/results_redis_${4}_${5}/outtext.txt
./../../raft-rl-test redisraft-rm $1 -e $2 --horizon $3 --cpuprofile cpu.prof --memprofile mem.prof --save /local/aborgare/results_redis_${4}_${5} 2>&1 | tee /local/aborgare/results_redis_${4}_${5}/outtext.txt

# output and kill open processes
pkill -9 redis-server
pkill -9 raft-rl-test

# clean the results folder and copy from local to RSE
rm -rf /RSE/RLDS/work/aborgare/data/results_redis_${4}_${5}
cp -r /local/aborgare/results_redis_${4}_${5} /RSE/RLDS/work/aborgare/data/results_redis_${4}_${5}

# mkdir ./results_redis_${4}_${5}
# ./raft-rl-test redisraft-rm $1 -e $2 --horizon $3 --save ./results_redis_${4}_${5} 2>&1 | tee ./results_redis_${4}_${5}/outtext.txt