#!/bin/bash
# ptrecon.sh

for i in {1..255}; do
  {
    resp=$(dig +short -x "172.16.10.$i")
    if [ -n "$resp" ]; then
      echo "172.16.10.$i -> $resp"
    fi
  } &
done
wait