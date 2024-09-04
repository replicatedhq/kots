#!/bin/bash

set -e

component=$1

# Check if a component name was provided
if [ -z "$component" ]; then
	echo "Error: No component name provided."
	exit 1
fi

# Check if already up
if [ -f "./dev/patches/$component-down.yaml.tmp" ]; then
  echo "Error: already up, run 'make $component-down' first."
  exit 1
fi

# Get component metadata
deployment=$(jq -r ".\"$component\".deployment" ./dev/metadata.json)

echo "Patching deployment..."

# Save current deployment state
kubectl get deployment $deployment -oyaml > ./dev/patches/$component-down.yaml.tmp

# Prepare and apply the patch
# The /host_mnt directory on Docker Desktop for macOS is a virtualized path that represents
# the mounted directories from the macOS host filesystem into the Docker Desktop VM.
# This is required for using HostPath volumes in Kubernetes.

sed "s|__PROJECT_DIR__|/host_mnt$(pwd)|g" ./dev/patches/$component-up.yaml > ./dev/patches/$component-up.yaml.tmp
kubectl patch deployment $component --patch-file ./dev/patches/$component-up.yaml.tmp
rm ./dev/patches/$component-up.yaml.tmp

# Wait for rollout to complete
kubectl rollout status deployment/$deployment

# Exec into the updated deployment
kubectl exec -it deployment/$deployment -- bash
