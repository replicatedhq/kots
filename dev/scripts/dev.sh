#!/bin/bash

set -e

. dev/scripts/common.sh

# Ensure kubectl context is orbstack
if [ $(kubectl config current-context) != "orbstack" ]; then
    echo "Error: kubectl context is not set to orbstack"
    exit 1
fi

function build_dev() {
  echo "Building $1..."
  populate $1
  build $1
  restart $1
  echo ""
}

build_dev kotsadm
build_dev kotsadm-web
build_dev kotsadm-migrations
build_dev kurl-proxy

kubectl apply -R -f dev/manifests

WITH_MINIO=${WITH_MINIO:-true}
if [ "${WITH_MINIO}" = "false" ]; then
  echo "Disabling Minio for snapshots; KOTS will use the Local Volume Provider path for HostPath/NFS destinations..."
  kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: kotsadm-confg
  labels:
    kots.io/kotsadm: "true"
    kots.io/backup: velero
data:
  minio-enabled-snapshots: "false"
EOF
else
  # Remove any previous dev toggle so the default (Minio enabled) is restored.
  kubectl delete configmap kotsadm-confg --ignore-not-found
fi

# kotsadm-web relies on host files to minimize the image size and
# to enable hot reloading by default, so it should always be "up".
render dev/patches/kotsadm-web-up.yaml | kubectl patch deployment kotsadm-web --patch-file=/dev/stdin
