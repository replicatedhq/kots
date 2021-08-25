
#!/usr/bin/env bash
#
# This is a replicated script.
#
# Syntax: ./k3s-debian.sh [k3s version] [k3s SHA256]

set -e

K3S_VERSION="${1:-"latest"}" # latest is also valid
K3S_SHA256="${2:-"automatic"}"

GPG_KEY_SERVERS="keyserver hkp://keyserver.ubuntu.com:80
keyserver hkps://keys.openpgp.org
keyserver hkp://keyserver.pgp.com"

architecture="$(uname -m)"
case $architecture in
    x86_64) architecture="amd64";;
    aarch64 | armv8*) architecture="arm64";;
    aarch32 | armv7* | armvhf*) architecture="armhf";;
    *) echo "(!) Architecture $architecture unsupported"; exit 1 ;;
esac

# Figure out correct version of a three part version number is not passed
find_version_from_git_tags() {
    local variable_name=$1
    local requested_version=${!variable_name}
    if [ "${requested_version}" = "none" ]; then return; fi
    local repository=$2
    local prefix=${3:-"tags/v"}
    local separator=${4:-"."}
    local last_part_optional=${5:-"false"}    
    if [ "$(echo "${requested_version}" | grep -o "." | wc -l)" != "2" ]; then
        local escaped_separator=${separator//./\\.}
        local last_part
        if [ "${last_part_optional}" = "true" ]; then
            last_part="(${escaped_separator}[0-9ks\+]+)?"
        else
            last_part="${escaped_separator}[0-9ks\+]+"
        fi
        local regex="${prefix}\\K[0-9]+${escaped_separator}[0-9]+${last_part}$"
        local version_list="$(git ls-remote --tags ${repository} | grep -oP "${regex}" | tr -d ' ' | tr "${separator}" "." | sort -rV)"
        echo $version_list
        if [ "${requested_version}" = "latest" ] || [ "${requested_version}" = "current" ] || [ "${requested_version}" = "lts" ]; then
            declare -g ${variable_name}="$(echo "${version_list}" | head -n 1)"
        else
            set +e
            declare -g ${variable_name}="$(echo "${version_list}" | grep -E -m 1 "^${requested_version//./\\.}([\\.\\s+]|$)")"
            set -e
        fi
    fi
    if [ -z "${!variable_name}" ] || ! echo "${version_list}" | grep "^${!variable_name//./\\.}$" > /dev/null 2>&1; then
        echo -e "Invalid ${variable_name} value: ${requested_version}\nValid values:\n${version_list}" >&2
        exit 1
    fi
    echo "${variable_name}=${!variable_name}"
}

# Install K3s, verify checksum
if [ "${K3S_VERSION}" != "none" ]; then
    echo "Downloading k3s..."
    urlPrefix=
    if [ "${K3S_VERSION}" = "latest" ] || [ "${K3S_VERSION}" = "lts" ] || [ "${K3S_VERSION}" = "current" ] || [ "${K3S_VERSION}" = "stable" ]; then
        K3S_VERSION="latest"
        urlPrefix="https://github.com/k3s-io/k3s/releases/latest/download"
    else
        find_version_from_git_tags K3S_VERSION https://github.com/k3s-io/k3s
        if [ "${K3S_VERSION::1}" != "v" ]; then
            K3S_VERSION="v${K3S_VERSION}"
        fi
        urlPrefix="https://github.com/k3s-io/k3s/releases/download/${K3S_VERSION}"
    fi
    
    # URL encode plus sign
    K3S_VERSION="$(echo $K3S_VERSION | sed --expression='s/+/%2B/g')"

    # latest is also valid in the download URLs
    downloadUrl="${urlPrefix}/k3s${architecture}"
    if [ "${architecture}" = "amd64" ]; then
        downloadUrl="${urlPrefix}/k3s"
    fi

    curl -sSL -o /usr/local/bin/k3s "${downloadUrl}"
    chmod 0755 /usr/local/bin/k3s

    if [ "$K3S_SHA256" = "automatic" ]; then

        shaUrl="${urlPrefix}/sha256sum-${architecture}.txt"
        if [ "${architecture}" = "armhf" ]; then
            shaUrl="${urlPrefix}/sha256sum-arm.txt"
        fi

        # Manifest contains image hashes, but we only need the binary
        K3S_SHA256="$(curl -sSL $shaUrl | grep -P '(^|\s)\Kk3s(?=\s|$)' | cut -d ' ' -f1 )"
    fi
    echo $K3S_SHA256
    ([ "${K3S_SHA256}" = "dev-mode" ] || (echo "${K3S_SHA256} */usr/local/bin/k3s" | sha256sum -c -))
    if ! type k3s > /dev/null 2>&1; then
        echo '(!) k3s installation failed!'
        exit 1
    fi
fi

echo -e "\nDone!"
