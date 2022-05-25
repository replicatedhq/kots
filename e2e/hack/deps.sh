#!/usr/bin/env sh

set -e

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

    export PATH=$INSTALL_DIR:$PATH

    tmpdir="$(mktemp -d)"
    cd $tmpdir

    mkdir -p $INSTALL_DIR

    EUID=$(id -u)
    EGID=$(id -g)

    curl -sLO "https://dl.k8s.io/release/$(curl -sL https://dl.k8s.io/release/stable.txt)/bin/$OS/$ARCH/kubectl" \
        && install -m 0755 kubectl $INSTALL_DIR/kubectl

    export K3D_INSTALL_DIR=$INSTALL_DIR
    curl -sL https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash -e && \
        ( [ $EUID -eq 0 -o "$USE_SUDO" != "true" ] || runAsRoot chown $EUID:$EGID $INSTALL_DIR/k3d )

    export HELM_INSTALL_DIR=$INSTALL_DIR
    curl -sL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash -e && \
        ( [ $EUID -eq 0 -o "$USE_SUDO" != "true" ] || runAsRoot chown $EUID:$EGID $INSTALL_DIR/helm )

    VELERO_RELEASE=$(curl -s "https://api.github.com/repos/vmware-tanzu/velero/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    curl -sLo velero.tar.gz "https://github.com/vmware-tanzu/velero/releases/download/$VELERO_RELEASE/velero-$VELERO_RELEASE-$OS-$ARCH.tar.gz" \
        && tar xzf velero.tar.gz \
        && install -m 0755 velero-*/velero $INSTALL_DIR/velero

    # curl -sL https://deb.nodesource.com/setup_18.x | runAsRoot bash -e \
    #     && runAsRoot apt-get install -y --no-install-recommends nodejs \
    #     && runAsRoot rm -rf /var/lib/apt/lists/* \
    #     && npm install --prefix $INSTALL_DIR @testim/testim-cli

    rm -rf $tmpdir
}

main
