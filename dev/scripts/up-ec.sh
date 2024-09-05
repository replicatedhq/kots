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

function docker_exec() {
  docker exec -it -w /replicatedhq/kots node0 $@
}

# The kotsadm dev image does not have a web component, and kotsadm-web service does not exist in embedded cluster.
# Deploy kotsadm-web service instead of shipping a web component in the kotsadm dev image so
# we can achieve a faster dev experience with hot reloading.
if [ "$component" == "kotsadm" || "$component" == "kotsadm-web" ]; then
  docker_exec k0s kubectl apply -f dev/manifests/kotsadm-web -n kotsadm
fi

# Save current deployment state
docker_exec k0s kubectl get deployment $deployment -n kotsadm -oyaml > ./dev/patches/$component-down-ec.yaml.tmp

# Prepare and apply the patch
render_ec dev/patches/$component-up.yaml | docker_exec k0s kubectl patch deployment $deployment -n kotsadm --patch-file=/dev/stdin

# Wait for rollout to complete
docker_exec k0s kubectl rollout status deployment/$deployment -n kotsadm

# Exec into the updated deployment
docker_exec k0s kubectl exec -it deployment/$deployment -n kotsadm -- bash
