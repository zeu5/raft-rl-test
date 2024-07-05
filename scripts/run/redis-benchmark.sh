#!/bin/bash

# runs $3 sequential experiments of redisraft benchmark. Processes the output folders afterwards.

# ARGS:
#   1 hierarchy set name - set7 or set8
#   2 comparison time limit: short, medium, std
#       (30m, 1h, 8h)
#   3 number of sequential iterations of the experiment

parent_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )
cd "$parent_path"

# delete and create output folder
rm -rf ${4}
mkdir -m 777 ${4}

cd ./../../ # move to main repo folder
mkdir -p -m 777 results
rm -rf results/redis-benchmark-$2
mkdir -m 777 results/redis-benchmark-$2

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

for (( i=1; i<=$3 ; i++ ))
do
    ./raft-rl-test redisraft-rm $1 -e 10000 --horizon 30 -t $2 --save results/redis-benchmark-$2/exp$i # 2>&1 | tee ${4}/outtext.txt

    sleep 3
    pkill -9 redis-server
    pkill -9 raft-rl-test
done

python3 ./scripts/graphs/coverage/createDataFolders.py results/redis-benchmark-$2

python3 ./scripts/graphs/coverage/plotAll.py results/redis-benchmark-$2
python3 ./scripts/graphs/coverage/customPlot.py results/redis-benchmark-$2/genericCoverageData

python3 ./scripts/graphs/coverage/analyzeRmData.py results/redis-benchmark-$2
