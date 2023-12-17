#!/bin/bash

set -eo pipefail

# this runs as part of pre-build

echo "on-create start"
echo "$(date +'%Y-%m-%d %H:%M:%S')    on-create start" >> "$HOME/status"

touch $HOME/.bashrc

echo "-----> Install k3d"

curl -s https://raw.githubusercontent.com/rancher/k3d/main/install.sh | bash
# k3d cluster create mycluster --config k3d.yaml
k3d cluster create mycluster --port '32000:32000' --port '30880:30880' --port '8800:8800'
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

curl -LO https://github.com/vmware-tanzu/velero/releases/download/v1.12.2/velero-v1.12.2-linux-amd64.tar.gz
tar zxvf velero-v1.12.2-linux-amd64.tar.gz
sudo mv velero-v1.12.2-linux-amd64/velero /usr/local/bin

echo "-----> Prepare cluster"

kubectl create ns test
make cache

echo "on-create complete"
echo "$(date +'%Y-%m-%d %H:%M:%S')    on-create complete" >> "$HOME/status"
