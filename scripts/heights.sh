#!/usr/bin/env bash
# Show the latest block height every node has, side by side, once a second.
#
#   bash scripts/heights.sh 26657 26654 26651 26648
#
# Pass the RPC ports the multi-node screen shows for each node. Nodes that
# are stopped show "down". Press Ctrl+C to stop.
#
# What you're looking for: running nodes agree on the height (within one
# block of each other) and it keeps climbing. If it stops climbing, fewer
# than two thirds of the stake is online and the chain has paused.

if [ "$#" -eq 0 ]; then
  echo "usage: bash scripts/heights.sh PORT1 PORT2 ..."
  exit 1
fi

while true; do
  line="$(date +%H:%M:%S) "
  i=1
  for port in "$@"; do
    h=$(curl -s --max-time 1 "http://127.0.0.1:${port}/status" \
        | sed -n 's/.*"latest_block_height":"\([0-9]*\)".*/\1/p')
    if [ -z "$h" ]; then h="down"; fi
    line+="node${i}:${h}  "
    i=$((i+1))
  done
  echo "$line"
  sleep 1
done
