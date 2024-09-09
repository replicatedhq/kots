#!/bin/bash

set -e

. dev/scripts/common.sh

# Ensure kubectl context is docker-desktop
if [ $(kubectl config current-context) != "docker-desktop" ]; then
    echo "Error: kubectl context is not set to docker-desktop"
    exit 1
fi

function build_dev() {
  echo "Building $1..."
  docker build -t $(image $1) -f $(dockerfile $1) $(dockercontext $1)
  restart $1
  echo ""
}

build_dev kotsadm
build_dev kotsadm-web
build_dev kotsadm-migrations
build_dev kurl-proxy

kubectl apply -R -f dev/manifests

# patch kotsadm-web to enable hot reloading
render dev/patches/kotsadm-web-up.yaml | kubectl patch deployment kotsadm-web --patch-file=/dev/stdin
