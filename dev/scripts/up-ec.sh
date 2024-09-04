#!/bin/bash

set -e

DEPLOYMENT=$1

# Check if a deployment name was provided
if [ -z "$DEPLOYMENT" ]; then
	echo "Error: No deployment name provided."
	exit 1
fi

# Check if already up
if [ -f "./dev/patches/$DEPLOYMENT-down.yaml.tmp" ]; then
  echo "Error: already up, run 'make $DEPLOYMENT-down-ec' first."
  exit 1
fi

# TODO NOW: get image from deployment name
if docker images | grep -q kotsadm-api-dev; then
  echo "kotsadm-api-dev image already exists, skipping build..."
else
  docker build -t kotsadm-api-dev -f ./hack/dev/skaffold.Dockerfile .
fi

docker exec node0 k0s ctr images ls | grep kotsadm-api-dev
echo $?

# TODO NOW: get image from deployment name
if docker exec node0 k0s ctr images ls | grep -q kotsadm-api-dev; then
  echo "kotsadm-api-dev image already loaded in embedded cluster, skipping import..."
else
  echo "Loading image into embedded cluster..."
  docker save kotsadm-api-dev | docker exec -i node0 k0s ctr images import -
fi

echo "Patching deployment in embedded cluster..."

function docker_exec() {
    docker exec -it -w /replicatedhq/kots node0 $@
}

# Save current deployment state
docker_exec k0s kubectl get deployment $DEPLOYMENT -n kotsadm -oyaml > ./dev/patches/$DEPLOYMENT-down.yaml.tmp

# Prepare and apply the patch
docker_exec sed 's|__PROJECT_DIR__|/replicatedhq/kots|g' ./dev/patches/$DEPLOYMENT-up.yaml > ./dev/patches/$DEPLOYMENT-up.yaml.tmp
docker_exec k0s kubectl patch deployment $DEPLOYMENT -n kotsadm --patch-file ./dev/patches/$DEPLOYMENT-up.yaml.tmp

# Clean up temporary file
docker_exec rm ./dev/patches/$DEPLOYMENT-up.yaml.tmp

# Wait for rollout to complete
docker_exec k0s kubectl rollout status deployment/$DEPLOYMENT -n kotsadm

# Exec into the updated deployment
docker_exec k0s kubectl exec -it deployment/$DEPLOYMENT -n kotsadm -- bash
