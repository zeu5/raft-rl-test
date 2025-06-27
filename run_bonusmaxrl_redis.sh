#!/bin/bash

if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <num_iterations>"
    exit 1
fi

iterations=$1

if [ -d "results" ]; then
    rm -rf results
fi

results_path="results"

pgrep redis-server
pgrep raft-rl-test

pkill -9 redis-server
pkill -9 raft-rl-test

sleep 3

./raft-rl-test redisraft-rm expl-rl -e $1 --horizon 25 --save $results_path 2>&1 | tee $results_path/outtext.txt
