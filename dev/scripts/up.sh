#!/bin/bash

set -e

. dev/scripts/common.sh

component=$1

# Check if a component name was provided
if [ -z "$component" ]; then
	echo "Error: No component name provided."
	exit 1
fi

# Get component metadata
deployment=$(jq -r ".\"$component\".deployment" dev/metadata.json)

# Check if already up
if [ -f "dev/patches/$component-down.yaml.tmp" ]; then
  up $deployment
  exit 0
fi

# Save current state
kubectl get deployment $deployment -oyaml > dev/patches/$component-down.yaml.tmp

# Prepare and apply the patch
render dev/patches/$component-up.yaml | kubectl patch deployment $deployment --patch-file=/dev/stdin

# Wait for rollout to complete
kubectl rollout status deployment/$deployment

# Up into the updated deployment
up $deployment
