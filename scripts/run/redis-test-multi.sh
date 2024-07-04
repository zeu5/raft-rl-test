#!/bin/bash

# runs a short initial test for redis with 3 consecutive runs. processes the output and generates plots and json summaries.

parent_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )
cd "$parent_path"

# delete and create output folder
rm -rf ${4}
mkdir -m 777 ${4}

cd ./../../ # move to main repo folder
mkdir -p -m 777 results
rm -rf results/init-test-multi
mkdir -m 777 results/init-test-multi

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
./raft-rl-test redisraft-rm set0 -e 50 --horizon 20 --save results/init-test-multi/exp1 # 2>&1 | tee ${4}/outtext.txt

sleep 3
pkill -9 redis-server
pkill -9 raft-rl-test

./raft-rl-test redisraft-rm set0 -e 50 --horizon 20 --save results/init-test-multi/exp2 # 2>&1 | tee ${4}/outtext.txt

sleep 3
pkill -9 redis-server
pkill -9 raft-rl-test

./raft-rl-test redisraft-rm set0 -e 50 --horizon 20 --save results/init-test-multi/exp3 # 2>&1 | tee ${4}/outtext.txt

sleep 3
pkill -9 redis-server
pkill -9 raft-rl-test

python3 ./scripts/graphs/coverage/createDataFolders.py results/init-test-multi

python3 ./scripts/graphs/coverage/plotAll.py results/init-test-multi
python3 ./scripts/graphs/coverage/customPlot.py results/init-test-multi/genericCoverageData

python3 ./scripts/graphs/coverage/analyzeRmData.py results/init-test-multi
