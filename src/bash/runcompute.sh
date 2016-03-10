#!/bin/bash

# current working directory
directory="../"
# Executable
executable="greencompute.py";

# compute resource node
node=$1
# compute port
port=$2

# boot compute_resource
echo "Booting compute_resource on $node:$port"
echo $(ssh $node bash -c "'python $directory$executable --port $port'")

#ssh $node bash -c "'pkill greencompute.py'"
