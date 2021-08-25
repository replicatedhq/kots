#!/usr/bin/env bash

# k3d
curl -s "https://raw.githubusercontent.com/rancher/k3d/main/install.sh" | bash 

# skaffold
curl -Lo /tmp/skaffold "https://storage.googleapis.com/skaffold/releases/latest/skaffold-linux-amd64"
sudo install /tmp/skaffold /usr/local/bin/

# kustomize
pushd /tmp
curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"  | bash
popd
sudo mv /tmp/kustomize /usr/local/bin/ 


