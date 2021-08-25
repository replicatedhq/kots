#!/usr/bin/env bash

# Setup Skaffold
cp /etc/replicated/skaffold.config $HOME/.skaffold/config

# Setup the cluster
k3d cluster create --config /etc/replicated/k3d-cluster.yaml --kubeconfig-update-default

# install Krew 
# TODO (dans): ditch krew and just download the latest binaries on the path in Dockerfile
(
  set -x; cd "$(mktemp -d)" &&
  OS="$(uname | tr '[:upper:]' '[:lower:]')" &&
  ARCH="$(uname -m | sed -e 's/x86_64/amd64/' -e 's/\(arm\)\(64\)\?.*/\1\2/' -e 's/aarch64$/arm64/')" &&
  curl -fsSLO "https://github.com/kubernetes-sigs/krew/releases/latest/download/krew.tar.gz" &&
  tar zxvf krew.tar.gz &&
  KREW=./krew-"${OS}_${ARCH}" &&
  "$KREW" install krew
)

export PATH="${KREW_ROOT:-$HOME/.krew}/bin:$PATH"

# install krew plugins
kubectl krew install schemahero
kubectl krew install support-bundle
kubectl krew install preflights
kubectl krew install view-secret

# install schemahero in the cluster
kubectl schemahero install

k3d cluster stop replicated

# Make the cache
make cache

# Clone any extra repos here

# Autocomplete Kubernetes
cat >> ~/.zshrc << EOF

source <(kubectl completion zsh)
alias k=kubectl
complete -F __start_kubectl k
EOF

# Set Git Editor Preference
cat >> ~/.zshrc << EOF

export VISUAL=vim
export EDITOR="$VISUAL"
EOF
