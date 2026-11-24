#!/usr/bin/env bash
set -euo pipefail

[[ "$GITHUB_REPOSITORY" == replicatedhq/kots && "$GITHUB_EVENT_NAME" == workflow_dispatch ]]
[[ "$LEGACY_COMPANION" == standalone-dex-gomplate ]]
[[ -z "$LEGACY_KOTS_VERSION" && -z "$LEGACY_ARCHIVE_DIGEST" && -z "$LEGACY_OVERLAY_RECEIPT" ]]

repo=index.docker.io/kotsadm/dex
tag=2.44.0
original=sha256:a0b04348b0ca5d7abc686705ef034a1491edb7427e51d88d7a042b67be2144e0
original_amd64=sha256:ebdbf7a04fc354d68676b617aa7c5069f7151f8afae3aa806b48e559410edc96
candidate=sha256:7870f204372ff6d32e9b9b29807454c5b7dc035f0899985f4d89feae54c8ac88
candidate_amd64=sha256:60fd3d4fdd2afe6b1530b4223cda5146ae8a6db57ed1c2847056e69ec2bfe009
archive_sha=sha256:f93c6373dde0d671710e7fb9d1cf3d890ffc826d8caf2a410e108d00666f8d69
archive_size=111618904
release_id=457394462
asset_id=789046583
asset=dex-2.44.0-baseline-carrier.tar.gz
work="$(mktemp -d)"
trap 'rm -rf -- "$work"' EXIT

gh api "repos/$GITHUB_REPOSITORY/releases/$release_id" |
    jq -e --argjson asset_id "$asset_id" --argjson size "$archive_size" --arg sha "$archive_sha" --arg asset "$asset" '
      .draft and .tag_name == "kots-dex-baseline-carrier-20261124" and (.assets | length == 1) and
      any(.assets[]; .id == $asset_id and .name == $asset and .state == "uploaded" and .size == $size and .digest == $sha)
    ' >/dev/null
gh api --header 'Accept: application/octet-stream' \
    "repos/$GITHUB_REPOSITORY/releases/assets/$asset_id" > "$work/$asset"
printf '%s  %s\n' "${archive_sha#sha256:}" "$work/$asset" | sha256sum -c -
python3 - "$work/$asset" "$work" <<'PY'
from pathlib import Path
import sys
import tarfile

with tarfile.open(Path(sys.argv[1]), "r:gz") as archive:
    archive.extractall(Path(sys.argv[2]), filter="data")
PY
grep -Fq 'passed native entrypoint, templating, SQLite and OIDC discovery on baseline x86-64.' \
    "$work/baseline-overlay-runtime.log"

skopeo inspect --raw "docker://$repo@$original" > "$work/original-index.json"
skopeo inspect --raw "oci:$work/baseline-overlay:candidate" > "$work/candidate-index.json"
printf '%s  %s\n' "${original#sha256:}" "$work/original-index.json" | sha256sum -c -
printf '%s  %s\n' "${candidate#sha256:}" "$work/candidate-index.json" | sha256sum -c -
non_amd64='.manifests | map(select(.platform.os != "linux" or .platform.architecture != "amd64"))'
diff -u <(jq -S "$non_amd64" "$work/original-index.json") <(jq -S "$non_amd64" "$work/candidate-index.json")
skopeo copy --preserve-digests "docker://$repo@$original_amd64" "oci:$work/original-amd64:original"
jq -ce --arg original "$original" --arg original_amd64 "$original_amd64" \
    --arg index "$candidate" --arg amd64 "$candidate_amd64" '
    select(.original_index == $original and .original_amd64 == $original_amd64 and .index == $index and .amd64 == $amd64) |
    select(.files == {"usr/bin/gomplate":{
        "old":"sha256:ff07bff911624b74f752abf1cb9fd92286c23c0277e306d956c67a32148559ec",
        "new":"sha256:ca559988c6a5a0f5a2ecb8fd97ec22bd059902b7fd20b6847f1ed2a13889e586"}}) |
    . + {candidate_index: .index, candidate_amd64: .amd64}
' "$work/baseline-overlay-receipt.json" > "$work/receipt.json"

sbom=(--companion dex --original-oci-layout "$work/original-amd64")
if cosign download sbom "$repo@$original" > "$work/original-index.spdx.json" 2> "$work/cosign.log"; then
    sbom+=(--original-index-spdx "$work/original-index.spdx.json")
elif ! grep -Fqx -e 'Error: no SBOM found attached to image index' \
                -e 'Error: no sbom attached to reference' "$work/cosign.log"; then
    cat "$work/cosign.log" >&2
    exit 1
fi
python3 scripts/rebind-legacy-image-sbom.py "$work/receipt.json" "$work/baseline-overlay" "$work/sbom" "${sbom[@]}"
SYFT_CHECK_FOR_APP_UPDATE=false "$SYFT_BIN" scan "oci-dir:$work/baseline-overlay" \
    --platform linux/amd64 -o spdx-json > "$work/sbom/amd64.spdx.json"
jq -e '.spdxVersion == "SPDX-2.3" and (.packages | length > 0)' "$work/sbom/amd64.spdx.json" >/dev/null

current_digest() {
    skopeo inspect --raw "$@" "docker://$repo:$tag" | sha256sum | awk '{print "sha256:"$1}'
}

live="$(current_digest)"
[[ "$live" == "$original" || "$live" == "$candidate" ]]
printf 'Verified %s:%s, its original ARM64 descriptor, and any historical SBOM attachments.\n' "$repo" "$tag"
if [[ "$LEGACY_PUBLISH" != true ]]; then
    exit 0
fi

: "${DOCKERHUB_USER:?}"
: "${DOCKERHUB_PASSWORD:?}"
export DOCKER_CONFIG="$work/auth"
install -d -m 700 "$DOCKER_CONFIG"
printf '%s' "$DOCKERHUB_PASSWORD" | docker login --username "$DOCKERHUB_USER" --password-stdin
unset DOCKERHUB_PASSWORD
auth="$DOCKER_CONFIG/config.json"
skopeo copy --all --preserve-digests --dest-authfile "$auth" \
    "oci:$work/baseline-overlay:candidate" "docker://$repo@$candidate"
for binding in "$candidate_amd64:amd64" "$candidate:index"; do
    subject="${binding%:*}"
    document="$work/sbom/${binding##*:}.spdx.json"
    if [[ -f "$document" ]]; then
        cosign attach sbom --type spdx --sbom "$document" "$repo@$subject"
        cosign download sbom "$repo@$subject" | jq -cS . | cmp - <(jq -cS . "$document")
    fi
done
live="$(current_digest --authfile "$auth")"
[[ "$live" == "$original" || "$live" == "$candidate" ]]
if [[ "$live" == "$original" ]]; then
    skopeo copy --all --preserve-digests --authfile "$auth" "docker://$repo@$candidate" "docker://$repo:$tag"
fi
for attempt in {1..10}; do
    if [[ "$(current_digest --authfile "$auth")" == "$candidate" ]]; then
        printf 'Verified %s:%s at %s.\n' "$repo" "$tag" "$candidate"
        exit 0
    fi
    sleep 3
done
exit 1
