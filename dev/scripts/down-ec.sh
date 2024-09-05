#!/bin/bash

set -e

component=$1

# Check if a deployment name was provided
if [ -z "$component" ]; then
	echo "Error: No component name provided."
	exit 1
fi

# Check if already down
if [ ! -f "dev/patches/$component-down-ec.yaml.tmp" ]; then
  echo "Error: already down."
  exit 1
fi

echo "Reverting..."

function docker_exec() {
  docker exec -it -w /replicatedhq/kots node0 $@
}

if [ "$component" == "kotsadm" ] || [ "$component" == "kotsadm-web" ]; then
  docker_exec k0s kubectl delete -f dev/manifests/kotsadm-web -n kotsadm
fi

docker_exec k0s kubectl replace -f dev/patches/$component-down-ec.yaml.tmp --force
docker_exec rm dev/patches/$component-down-ec.yaml.tmp
