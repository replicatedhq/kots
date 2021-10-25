#!/bin/bash

if [ -z ${KGRID_API_TOKEN} ]; then
    echo "KGRID_API_TOKEN must be set"
    exit 1
fi

if [ -z ${KGRID_RUN_ID} ]; then
    echo "KGRID_RUN_ID must be set"
    exit 1
fi

touch -d '+40 minute' kgrid_timeout
while true; do
  curl -s https://kgrid.replicated.systems/v1/run/${KGRID_RUN_ID}/outcome \
    -H "Authorization: $KGRID_API_TOKEN" \
    > outcome.json

  RUN_RESULT=`cat outcome.json | jq ".outcome"`

  if [ "$RUN_RESULT" == "\"Pass\"" ]; then
    echo "Passed"
    cat outcome.json | jq .
    exit 0
  fi

  if [ "$RUN_RESULT" == "\"Fail\"" ]; then
    echo "Failed"
    cat outcome.json | jq .
    exit 1
  fi

  touch kgrid_now
  if test kgrid_now -nt kgrid_timeout; then
    echo "Timed out"
    cat outcome.json | jq .
    exit 1
  fi

  sleep 30
done
