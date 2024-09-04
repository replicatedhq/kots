#!/bin/bash

set -e

DEPLOYMENT=$1

# Check if a deployment name was provided
if [ -z "$DEPLOYMENT" ]; then
	echo "Error: No deployment name provided."
	exit 1
fi

# Check if already down
if [ ! -f "./dev/patches/$DEPLOYMENT-down.yaml.tmp" ]; then
  echo "Error: already down, run 'make $DEPLOYMENT-up-ec' first."
  exit 1
fi

echo "Reverting deployment in embedded cluster..."

function docker_exec() {
    docker exec -it -w /replicatedhq/kots node0 $@
}

docker_exec k0s kubectl replace -f ./dev/patches/$*-down.yaml.tmp --force
docker_exec rm ./dev/patches/$*-down.yaml.tmp
