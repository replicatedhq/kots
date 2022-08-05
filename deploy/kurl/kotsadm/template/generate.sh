#!/bin/bash

set -euo pipefail

function require() {
   if [ -z "$2" ]; then
       echo "validation failed: $1 unset"
       exit 1
   fi
}

# From Client Payload
require KOTSADM_VERSION "${KOTSADM_VERSION}"

function generate() {
    local kotsadm_tag=$1
    local kotsadm_dir=$2
    local kotsadm_binary_version=$3
    local dir="../${kotsadm_dir}"\

    if [ -d "$dir" ]; then
        echo "Kotsadm ${kotsadm_dir} add-on already exists"
        
        # Clean out the directory in case the template has removed any files
        rm -rf "$dir"
    fi
    mkdir -p "$dir"

    cp -r base/* "$dir/"
    find "$dir" -type f -exec sed -i -e "s/__KOTSADM_TAG__/$kotsadm_tag/g" {} \;
    find "$dir" -type f -exec sed -i -e "s/__KOTSADM_DIR__/$kotsadm_dir/g" {} \;
    find "$dir" -type f -exec sed -i -e "s/__KOTSADM_BINARY_VERSION__/$kotsadm_binary_version/g" {} \;

    # grab generated dot env file containing the latest version tags, export environment variables in dot env file
    # and update manifest with latest image tags
    export $(cat ../../../../.image.env | sed 's/#.*//g' | xargs)
    find "$dir" -type f -exec sed -i -e "s/__POSTGRES_10_TAG__/$POSTGRES_10_TAG/g" {} \;
    find "$dir" -type f -exec sed -i -e "s/__POSTGRES_14_TAG__/$POSTGRES_14_TAG/g" {} \;
    sed -i -e "s/__DEX_TAG__/$DEX_TAG/g" "${dir}/Manifest"

}

function main() {
    export KOTSADM_VERSION=${KOTSADM_VERSION#v}

    echo "generating v$KOTSADM_VERSION version"
    generate "v$KOTSADM_VERSION" "$KOTSADM_VERSION" "v$KOTSADM_VERSION"
}

main "$@"
