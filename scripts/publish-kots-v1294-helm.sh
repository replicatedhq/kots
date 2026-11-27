#!/usr/bin/env bash
set -euo pipefail
umask 077

[[ "$GITHUB_REPOSITORY" == replicatedhq/kots && "$GITHUB_EVENT_NAME" == workflow_dispatch ]]
[[ "$EXPECTED_INDEX" =~ ^sha256:[0-9a-f]{64}$ ]]
repo=index.docker.io/kotsadm/kotsadm
tag=v1.129.4
original=sha256:04d5faa4ad52e929f026a0d53fa9d073dd9ef2b822e37011896cdb46a02c1e11
work="${RESTORE_WORK:?}"
candidate="oci:${work}/candidate:candidate"

current_digest() {
  skopeo inspect --raw "$@" "docker://$repo:$tag" | sha256sum | awk '{print "sha256:" $1}'
}

check_candidate() {
  skopeo inspect --raw "$candidate" | sha256sum -c <(printf '%s  -\n' "${EXPECTED_INDEX#sha256:}")
}

case "${1:-}" in
  verify)
    live="$(current_digest)"
    [[ "$live" == "$original" || "$live" == "$EXPECTED_INDEX" ]]
    mkdir -m 700 "$work"
    skopeo copy --all --preserve-digests "docker://$repo@$original" "oci:${work}/original:original"
    for arch in amd64 arm64; do
      asset="helm-v3.20.0-linux-$arch.tar.gz"
      curl -fsSL --proto '=https' --max-time 120 "https://get.helm.sh/$asset" -o "$work/$asset"
    done
    python3 scripts/restore-kots-v1294-helm.py "$work/original" "$work" "$work/candidate" > "$work/receipt.json"
    check_candidate
    cat "$work/receipt.json"
    for arch in amd64 arm64; do
      SYFT_CHECK_FOR_APP_UPDATE=false "$SYFT_BIN" scan "oci-dir:$work/candidate" \
        --platform "linux/$arch" -o spdx-json > "$work/$arch.spdx.json"
      jq -e '.spdxVersion == "SPDX-2.3" and any(.packages[]; .name == "helm.sh/helm/v3" and (.versionInfo == "v3.20.0" or .versionInfo == "3.20.0"))' "$work/$arch.spdx.json" >/dev/null
    done
    ;;
  publish)
    check_candidate
    export DOCKER_CONFIG="$work/auth"
    install -d -m 700 "$DOCKER_CONFIG"
    printf '%s' "$DOCKERHUB_PASSWORD" | docker login --username "$DOCKERHUB_USER" --password-stdin
    unset DOCKERHUB_PASSWORD
    auth="$DOCKER_CONFIG/config.json"
    live="$(current_digest --authfile "$auth")"
    [[ "$live" == "$original" || "$live" == "$EXPECTED_INDEX" ]]
    skopeo copy --all --preserve-digests --dest-authfile "$auth" "$candidate" "docker://$repo@$EXPECTED_INDEX"
    for arch in amd64 arm64; do
      digest="$(jq -er --arg arch "$arch" '.platforms[$arch].manifest' "$work/receipt.json")"
      cosign attach sbom --type spdx --sbom "$work/$arch.spdx.json" "$repo@$digest"
      cosign download sbom "$repo@$digest" | jq -cS . | cmp - <(jq -cS . "$work/$arch.spdx.json")
    done
    live="$(current_digest --authfile "$auth")"
    [[ "$live" == "$original" || "$live" == "$EXPECTED_INDEX" ]]
    if [[ "$live" == "$original" ]]; then
      skopeo copy --all --preserve-digests --authfile "$auth" "docker://$repo@$EXPECTED_INDEX" "docker://$repo:$tag"
    fi
    for attempt in {1..10}; do
      if [[ "$(current_digest --authfile "$auth")" == "$EXPECTED_INDEX" ]]; then
        printf 'Verified %s:%s at %s.\n' "$repo" "$tag" "$EXPECTED_INDEX"
        exit 0
      fi
      sleep 3
    done
    exit 1
    ;;
  *) exit 2 ;;
esac
