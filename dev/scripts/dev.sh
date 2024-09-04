#!/bin/bash

set -e

function image() {
  jq -r ".\"$1\".image" ./dev/metadata.json
}

function dockerfile() {
  jq -r ".\"$1\".dockerfile" ./dev/metadata.json
}

function dockercontext() {
  jq -r ".\"$1\".dockercontext" ./dev/metadata.json
}

function deployment() {
  jq -r ".\"$1\".deployment" ./dev/metadata.json
}

function restart() {
  if [ "$1" == "kotsadm-migrations" ]; then
    kubectl delete job $1 --ignore-not-found
  elif kubectl get deployment $(deployment $1) &>/dev/null; then
    kubectl rollout restart deployment/$(deployment $1)
  fi
}

function build() {
  echo "Building $1..."
  docker build -t $(image $1) -f $(dockerfile $1) $(dockercontext $1)
  restart $1
  echo ""
}

build kotsadm
build kotsadm-web
build kotsadm-migrations
build kurl-proxy

kubectl apply -k ./kustomize/overlays/dev
