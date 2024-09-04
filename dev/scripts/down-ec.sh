#!/bin/bash

set -e

component=$1

# Check if a deployment name was provided
if [ -z "$component" ]; then
	echo "Error: No deployment name provided."
	exit 1
fi

# Check if already down
if [ ! -f "./dev/patches/$component-down-ec.yaml.tmp" ]; then
  echo "Error: already down, run 'make $component-up-ec' first."
  exit 1
fi

echo "Reverting deployment in embedded cluster..."

function docker_exec() {
  docker exec -it -w /replicatedhq/kots node0 $@
}

docker_exec k0s kubectl replace -f ./dev/patches/$component-down-ec.yaml.tmp --force
docker_exec rm ./dev/patches/$component-down-ec.yaml.tmp
