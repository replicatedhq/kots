#!/bin/bash

set -e

. dev/scripts/common.sh

component=$1

# Check if a component name was provided
if [ -z "$component" ]; then
	echo "Error: No component name provided."
	exit 1
fi

# Check if already up
if [ -f "dev/patches/$component-down-ec.yaml.tmp" ]; then
  up_ec $component
  exit 0
fi

# Build and load the image into the embedded cluster
build_and_load_ec "$component"

# The kotsadm dev image does not have a web component, and kotsadm-web service does not exist in embedded cluster.
# Deploy kotsadm-web service instead of shipping a web component in the kotsadm dev image so
# we can achieve a faster dev experience with hot reloading.
if [ "$component" == "kotsadm" ] || [ "$component" == "kotsadm-web" ]; then
  build_and_load_ec "kotsadm-web"
  exec_ec k0s kubectl apply -f dev/manifests/kotsadm-web -n kotsadm
  patch_ec "kotsadm-web"
fi

# Save current deployment state
exec_ec k0s kubectl get deployment $(deployment $component) -n kotsadm -oyaml > dev/patches/$component-down-ec.yaml.tmp

# Patch the deployment
patch_ec $component

# Wait for rollout to complete
exec_ec k0s kubectl rollout status deployment/$(deployment $component) -n kotsadm

# Up into the updated deployment
up_ec $component
