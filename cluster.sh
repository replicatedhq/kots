#!/bin/bash

set -eo pipefail

echo "-----> Check if Docker is running"

docker_is_ready() {
    docker info > /dev/null 2>&1
}

if ! docker_is_ready; then
  echo "Docker isn't running!"
  exit 1
fi

echo "-----> Install kind"

go install sigs.k8s.io/kind@latest
if ! kind get clusters | grep -q dev-cluster; then
  kind create cluster --name dev-cluster --config kind.yaml --wait 60s
fi

echo "-----> Install Registry"

helm repo add twuni https://helm.twun.io
helm upgrade -i --create-namespace --namespace docker-registry docker-registry twuni/docker-registry \
  --set service.type=NodePort \
  --set service.nodePort=32000

echo "-----> Install kustomize"

KUSTOMIZE_VERSION=v5.3.0
gh release download "kustomize/$KUSTOMIZE_VERSION" --repo kubernetes-sigs/kustomize --pattern '*_darwin_arm64.tar.gz' --output /tmp/kustomize.tar.gz --clobber
tar -xf /tmp/kustomize.tar.gz -O > /tmp/kustomize
rm /tmp/kustomize.tar.gz
chmod +x /tmp/kustomize
mv /tmp/kustomize /usr/local/bin

echo "-----> Install & Configure skaffold"

SKAFFOLD_VERSION=v2.9.0
curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/"$SKAFFOLD_VERSION"/skaffold-darwin-arm64
chmod +x skaffold
mv skaffold /usr/local/bin

mkdir -p "$HOME/.skaffold"
cat <<EOF > "$HOME/.skaffold/config"
global:
  default-repo: localhost:32000
  kind-disable-load: true
kubeContexts: []
EOF

echo "-----> Install Velero"

VELERO_VERSION=v1.12.2
gh release download "$VELERO_VERSION" --repo vmware-tanzu/velero --pattern '*-darwin-arm64.tar.gz' --output /tmp/velero.tar.gz --clobber
tar -xzvf /tmp/velero.tar.gz -C /tmp
chmod +x /tmp/velero-$VELERO_VERSION-darwin-arm64/velero
mv /tmp/velero-$VELERO_VERSION-darwin-arm64/velero /usr/local/bin
rm /tmp/velero.tar.gz
rm -rf /tmp/velero-$VELERO_VERSION-darwin-arm64

velero install \
  --no-default-backup-location \
  --no-secret \
  --use-node-agent \
  --uploader-type=restic \
  --use-volume-snapshots=false \
  --plugins velero/velero-plugin-for-aws:v1.8.2,velero/velero-plugin-for-gcp:v1.8.2,velero/velero-plugin-for-microsoft-azure:v1.8.2,replicated/local-volume-provider:v0.6.7
