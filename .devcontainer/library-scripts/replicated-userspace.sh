#!/usr/bin/env bash

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

# install krew plugins
kubectl krew install schemahero
kubectl krew install support-bundle
kubectl krew install preflights
kubectl krew install view-secret

# Make the cache from master branch
pushd /tmp 
git clone https://github.com/replicatedhq/kots.git
pushd kots
# TODO (dans): find a way to cache images on image build
go mod download
popd
rm -rf kots
popd

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
