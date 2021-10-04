#!/usr/bin/env bash

# Setup the cluster
k3d cluster create --config /etc/replicated/k3d-cluster.yaml --kubeconfig-update-default

# install schemahero in the cluster
kubectl schemahero install

# Make the cache
make cache
skaffold build

# Clone any extra repos here


