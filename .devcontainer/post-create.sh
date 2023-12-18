#!/bin/bash

set -eo pipefail

# this runs at Codespace creation - not part of pre-build

echo "post-create start"
echo "$(date)    post-create start" >> "$HOME/status"

touch $HOME/.bashrc

echo "-----> Wait for docker daemon to start"

# Maximum number of attempts
MAX_ATTEMPTS=30
SLEEP_INTERVAL=1

docker_is_ready() {
    docker info > /dev/null 2>&1
}

attempt=0
until docker_is_ready || [ $attempt -eq $MAX_ATTEMPTS ]; do
    attempt=$((attempt+1))
    echo "Waiting for Docker daemon to start... Attempt $attempt"
    sleep $SLEEP_INTERVAL
done

if docker_is_ready; then
    echo "Docker daemon is ready!"
else
    echo "Timed out waiting for Docker daemon to start."
    exit 1
fi

echo "-----> Install k3d"

curl -s https://raw.githubusercontent.com/rancher/k3d/main/install.sh | bash
k3d cluster create mycluster --config .devcontainer/k3d.yaml
export KUBECONFIG="$(k3d kubeconfig write mycluster)"
echo "export KUBECONFIG=$KUBECONFIG" >> $HOME/.bashrc

echo "-----> Install Registry"

helm repo add twuni https://helm.twun.io
helm install --create-namespace --namespace docker-registry docker-registry twuni/docker-registry \
  --set service.type=NodePort \
  --set service.nodePort=32000

echo "-----> Install kustomize"

curl -L "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv4.5.7/kustomize_v4.5.7_linux_amd64.tar.gz" > /tmp/kustomize.tar.gz
tar -xf /tmp/kustomize.tar.gz -O > /tmp/kustomize
rm /tmp/kustomize.tar.gz
chmod a+x /tmp/kustomize
sudo mv /tmp/kustomize /usr/local/bin

echo "-----> Install & Configure skaffold"

curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/v1.19.0/skaffold-linux-amd64
chmod +x skaffold
sudo mv skaffold /usr/local/bin

mkdir -p $HOME/.skaffold
cat <<EOF > $HOME/.skaffold/config
global:
  default-repo: localhost:32000
kubeContexts: []
EOF

echo "-----> Install Velero"

curl -L https://github.com/vmware-tanzu/velero/releases/download/v1.12.2/velero-v1.12.2-linux-amd64.tar.gz > /tmp/velero.tar.gz
tar -xzvf /tmp/velero.tar.gz -C /tmp
chmod a+x /tmp/velero-v1.12.2-linux-amd64/velero
sudo mv /tmp/velero-v1.12.2-linux-amd64/velero /usr/local/bin
rm /tmp/velero.tar.gz
rm -rf /tmp/velero-v1.12.2-linux-amd64

velero install \
  --no-default-backup-location \
  --no-secret \
  --use-node-agent \
  --uploader-type=restic \
  --use-volume-snapshots=false \
  --plugins velero/velero-plugin-for-aws:v1.8.2,velero/velero-plugin-for-gcp:v1.8.2,velero/velero-plugin-for-microsoft-azure:v1.8.2,replicated/local-volume-provider:v0.5.6

echo "-----> Prepare cluster"

kubectl create ns test
make cache

echo "post-create complete"
echo "$(date +'%Y-%m-%d %H:%M:%S')    post-create complete" >> "$HOME/status"
