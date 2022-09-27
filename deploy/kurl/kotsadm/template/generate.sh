#!/bin/bash

set -euo pipefail

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
    # shellcheck disable=SC2046
    export $(grep -v '^#' ../../../../.image.env | xargs)
    find "$dir" -type f -exec sed -i -e "s/__POSTGRES_10_TAG__/$POSTGRES_10_TAG/g" {} \;
    find "$dir" -type f -exec sed -i -e "s/__POSTGRES_14_TAG__/$POSTGRES_14_TAG/g" {} \;
    sed -i -e "s/__DEX_TAG__/$DEX_TAG/g" "${dir}/Manifest"

}

function main() {
    local version="${1#v}"

    echo "generating v$version version"
    generate "v$version" "$version" "v$version"
}

main "$@"
