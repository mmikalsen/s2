#!/bin/bash

# compute resource node
node=$1

# shut down
ssh $node bash -c "'pkill greencompute.py'"
