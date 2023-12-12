#!/bin/bash

set -euo pipefail

function generate() {
    local kotsadm_version="$1"
    local kotsadm_image_registry="$2"
    local kotsadm_image_namespace="$3"
    local kotsadm_image_tag="$4"
    local kotsadm_binary="$5"

    local dir="../$kotsadm_version"

    # Clean out the directory in case the template has removed any files
    rm -rf "$dir"
    mkdir -p "$dir"

    cp -r base/* "$dir/"

    local kotsadm_image="$kotsadm_image_registry/$kotsadm_image_namespace/kotsadm:$kotsadm_image_tag"
    local kotsadm_migrations_image="$kotsadm_image_registry/$kotsadm_image_namespace/kotsadm-migrations:$kotsadm_image_tag"
    local kurl_proxy_image="$kotsadm_image_registry/$kotsadm_image_namespace/kurl-proxy:$kotsadm_image_tag"

    find "$dir" -type f -exec sed -i -e "s|__KOTSADM_IMAGE__|$kotsadm_image|g" {} \;
    find "$dir" -type f -exec sed -i -e "s|__KOTSADM_MIGRATIONS_IMAGE__|$kotsadm_migrations_image|g" {} \;
    find "$dir" -type f -exec sed -i -e "s|__KURL_PROXY_IMAGE__|$kurl_proxy_image|g" {} \;

    sed -i -e "s|__KOTSADM_BINARY__|$kotsadm_binary|g" "${dir}/Manifest"

    # The following environment variables will be exported by the .image.env file
    local rqlite_image="$kotsadm_image_registry/$kotsadm_image_namespace/rqlite:$RQLITE_TAG"
    find "$dir" -type f -exec sed -i -e "s|__RQLITE_IMAGE__|$rqlite_image|g" {} \;
    local dex_image="$kotsadm_image_registry/$kotsadm_image_namespace/dex:$DEX_TAG"
    find "$dir" -type f -exec sed -i -e "s|__DEX_IMAGE__|$dex_image|g" {} \;
}

DEFAULT_KOTSADM_IMAGE_REGISTRY=docker.io
DEFAULT_KOTSADM_IMAGE_NAMESPACE=kotsadm

function main() {
    local kotsadm_version="$1"
    local kotsadm_image_registry="${2:-$DEFAULT_KOTSADM_IMAGE_REGISTRY}"
    local kotsadm_image_namespace="${3:-$DEFAULT_KOTSADM_IMAGE_NAMESPACE}"
    local kotsadm_image_tag="${4:-}"
    local kotsadm_binary="${5:-}"

    if [ -z "$kotsadm_image_tag" ]; then
        kotsadm_image_tag="v$kotsadm_version"
    fi

    if [ -z "$kotsadm_binary" ]; then
        kotsadm_binary="https://github.com/replicatedhq/kots/releases/download/v$kotsadm_version/kots_linux_amd64.tar.gz"
    fi

    echo "Generating add-on version $kotsadm_version"
    generate "$kotsadm_version" "$kotsadm_image_registry" "$kotsadm_image_namespace" "$kotsadm_image_tag" "$kotsadm_binary"
    echo "Generating add-on version $kotsadm_version success"
}

main "$@"
