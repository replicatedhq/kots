#!/bin/bash

if [ -z ${KGRID_API_TOKEN} ]; then
    echo "KGRID_API_TOKEN must be set"
    exit 1
fi

if [ -z ${RUN_ID} ]; then
    echo "RUN_ID must be set"
    exit 1
fi

touch -d '+40 minute' kgrid_timeout
while true; do
  curl https://kgrid.replicated.systems/v1/run/${RUN_ID}/outcome \
    -H "Authorization: $KGRID_API_TOKEN" \
    > outcome.json

  RUN_RESULT=`cat outcome.json | jq ".outcome"`

  if [ "$RUN_RESULT" == "Pass" ]; then
    echo "Passed"
    cat outcome.json
    exit 0
  fi

  if [ "$RUN_RESULT" == "Fail" ]; then
    echo "Failed"
    cat outcome.json
    exit 1
  fi

  touch kgrid_now
  if test kgrid_now -nt kgrid_timeout; then
    echo "Timed out"
    cat outcome.json
    exit 1
  fi

  sleep 30
done
