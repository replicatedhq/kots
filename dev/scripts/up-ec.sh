#!/bin/bash

set -e

component=$1

# Check if a component name was provided
if [ -z "$component" ]; then
	echo "Error: No component name provided."
	exit 1
fi

# Check if already up
if [ -f "./dev/patches/$component-down-ec.yaml.tmp" ]; then
  echo "Error: already up, run 'make $component-down-ec' first."
  exit 1
fi

# Get component metadata
image=$(jq -r ".\"$component\".image" ./dev/metadata.json)
dockerfile=$(jq -r ".\"$component\".dockerfile" ./dev/metadata.json)
dockercontext=$(jq -r ".\"$component\".dockercontext" ./dev/metadata.json)
deployment=$(jq -r ".\"$component\".deployment" ./dev/metadata.json)

# Build the image
if docker images | grep -q "$image"; then
  echo "$image image already exists, skipping build..."
else
  docker build -t "$image" -f "$dockerfile" "$dockercontext"
fi

# Load the image into the embedded cluster
if docker exec node0 k0s ctr images ls | grep -q "$image"; then
  echo "$image image already loaded in embedded cluster, skipping import..."
else
  echo "Loading image into embedded cluster..."
  docker save "$image" | docker exec -i node0 k0s ctr images import -
fi

echo "Patching deployment in embedded cluster..."

function docker_exec() {
  docker exec -it -w /replicatedhq/kots node0 $@
}

# Save current deployment state
docker_exec k0s kubectl get deployment $deployment -n kotsadm -oyaml > ./dev/patches/$component-down-ec.yaml.tmp

# Prepare and apply the patch
# The embedded-cluster container mounts the KOTS project at /replicatedhq/kots
docker_exec sed 's|__PROJECT_DIR__|/replicatedhq/kots|g' ./dev/patches/$component-up.yaml > ./dev/patches/$component-up-ec.yaml.tmp
docker_exec k0s kubectl patch deployment $deployment -n kotsadm --patch-file ./dev/patches/$component-up-ec.yaml.tmp
docker_exec rm ./dev/patches/$component-up-ec.yaml.tmp

# Wait for rollout to complete
docker_exec k0s kubectl rollout status deployment/$deployment -n kotsadm

# Exec into the updated deployment
docker_exec k0s kubectl exec -it deployment/$deployment -n kotsadm -- bash
