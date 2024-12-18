#!/bin/bash

set -e

. dev/scripts/common.sh

component=$1

# Check if a component name was provided
if [ -z "$component" ]; then
  echo "Error: No component name provided."
  exit 1
fi

# kotsadm-web must already be up
if [ "$component" == "kotsadm-web" ]; then
  ec_up $component
  exit 0
fi

# Build and load the image into the embedded cluster
ec_build_and_load "$component"

# Use the dev image for kotsadm migrations
if [ "$component" == "kotsadm" ]; then
  ec_build_and_load "kotsadm-migrations" true
fi

# The kotsadm dev image does not have a web component, and kotsadm-web service does not exist in embedded cluster.
# Deploy kotsadm-web service instead of shipping a web component in the kotsadm dev image so
# we can achieve a faster dev experience with hot reloading.
if [ "$component" == "kotsadm" ]; then
  ec_build_and_load "kotsadm-web"
  ec_exec k0s kubectl --kubeconfig=/var/lib/embedded-cluster/k0s/pki/admin.conf apply -f dev/manifests/kotsadm-web -n kotsadm
  ec_patch "kotsadm-web"
fi

# Save original state
if [ ! -f "dev/patches/$component-down-ec.yaml.tmp" ]; then
  ec_exec k0s kubectl --kubeconfig=/var/lib/embedded-cluster/k0s/pki/admin.conf get deployment $(deployment $component) -n kotsadm -oyaml > dev/patches/$component-down-ec.yaml.tmp
fi

# Patch the deployment
ec_patch $component

# Wait for rollout to complete
ec_exec k0s kubectl --kubeconfig=/var/lib/embedded-cluster/k0s/pki/admin.conf rollout status deployment/$(deployment $component) -n kotsadm

# Up into the updated deployment
ec_up $component
