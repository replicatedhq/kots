#!/usr/bin/env bash

set -e
set -o pipefail
set -x

: ${USE_SUDO:="true"}
: ${INSTALL_DIR:="/usr/local/bin"}

# initArch discovers the architecture for this system.
initArch() {
  ARCH=$(uname -m)
  case $ARCH in
    armv5*) ARCH="armv5";;
    armv6*) ARCH="armv6";;
    armv7*) ARCH="arm";;
    aarch64) ARCH="arm64";;
    x86) ARCH="386";;
    x86_64) ARCH="amd64";;
    i686) ARCH="386";;
    i386) ARCH="386";;
  esac
}

# initOS discovers the operating system for this system.
initOS() {
  OS=$(uname|tr '[:upper:]' '[:lower:]')

  case "$OS" in
    # Minimalist GNU for Windows
    mingw*) OS='windows';;
  esac
}

runAsRoot() {
  if [ $EUID -ne 0 -a "$USE_SUDO" = "true" ]; then
    sudo "${@}"
  else
    "${@}"
  fi
}

main() {
    initArch
    initOS
    echo "OS=$OS, ARCH=$ARCH"

    export PATH=$INSTALL_DIR:$PATH

    tmpdir="$(mktemp -d)"
    cd $tmpdir

    mkdir -p $INSTALL_DIR

    curl -fsLO "https://dl.k8s.io/release/$(curl -sL https://dl.k8s.io/release/stable.txt)/bin/$OS/$ARCH/kubectl" \
        && install -m 0755 kubectl $INSTALL_DIR/kubectl

    export K3D_INSTALL_DIR=$INSTALL_DIR
    curl -sL https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash -e && \
        ( [ $(id -u) -eq 0 -o "$USE_SUDO" != "true" ] || runAsRoot chown $(id -u):$(id -g) $INSTALL_DIR/k3d )

    export HELM_INSTALL_DIR=$INSTALL_DIR
    curl -sL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash -e && \
        ( [ $(id -u) -eq 0 -o "$USE_SUDO" != "true" ] || runAsRoot chown $(id -u):$(id -g) $INSTALL_DIR/helm )

    VELERO_RELEASE=$(curl -s "https://api.github.com/repos/vmware-tanzu/velero/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    echo "VELERO_RELEASE=$VELERO_RELEASE"
    curl -fsLo velero.tar.gz "https://github.com/vmware-tanzu/velero/releases/download/$VELERO_RELEASE/velero-$VELERO_RELEASE-$OS-$ARCH.tar.gz" \
        && tar xzf velero.tar.gz \
        && install -m 0755 velero-*/velero $INSTALL_DIR/velero

    runAsRoot mkdir -p /etc/apt/keyrings
    curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | runAsRoot gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg
    NODE_MAJOR=18
    echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_$NODE_MAJOR.x nodistro main" | runAsRoot tee /etc/apt/sources.list.d/nodesource.list
    runAsRoot apt-get update && runAsRoot apt-get install nodejs -y

    rm -rf $tmpdir

    cd /playwright
    npm ci
    npx playwright install --with-deps
}

main
