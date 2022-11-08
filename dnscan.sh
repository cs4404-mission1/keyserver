#!/bin/bash

# Check for 2 args
if [ $# -ne 2 ]; then
  echo "Usage: $0 <domain> <wordlist>"
  exit 1
fi

# For line in file
while read -r line; do
  {
    resp=$(dig +short @10.64.10.2 "$line.$1")
    if [ -n "$resp" ]; then
      echo "$line.$1 -> $resp"
    fi
  } &
done <$2
wait
