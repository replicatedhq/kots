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

docker build -t $(image kotsadm) -f $(dockerfile kotsadm) $(dockercontext kotsadm)
docker build -t $(image kotsadm-web) -f $(dockerfile kotsadm-web) $(dockercontext kotsadm-web)
docker build -t $(image kotsadm-migrations) -f $(dockerfile kotsadm-migrations) $(dockercontext kotsadm-migrations)
docker build -t $(image kurl-proxy) -f $(dockerfile kurl-proxy) $(dockercontext kurl-proxy)

kubectl apply -k ./kustomize/overlays/dev
