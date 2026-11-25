#!/usr/bin/env bash
set -euo pipefail

[[ "$GITHUB_REPOSITORY" == replicatedhq/kots && "$GITHUB_EVENT_NAME" == workflow_dispatch ]]
[[ "$LEGACY_COMPANION" == standalone-kotsadm-kustomize ]]
[[ -z "$LEGACY_KOTS_VERSION" && -z "$LEGACY_ARCHIVE_DIGEST" && -z "$LEGACY_OVERLAY_RECEIPT" ]]

repo=index.docker.io/kotsadm/kotsadm
tag=latest
original=sha256:05ad673d98015c29f04abb16234af7e3e8d88917353026dcb2363377cb3b682d
original_amd64=sha256:19955f76ff142b75da500530f7ea2914aff353716a7f4ec9cea078a7aa576c87
candidate=sha256:fbbd56d17e39f462f21dab76b2f7da2fd8cac1d147e6be89fe08a9590e736b5d
candidate_amd64=sha256:9a91993a2e41085313925f93336bfd8c2368c7e1078d6074b6eab4b1af101fa8
archive_sha=sha256:19ace3d27b99c37485478161d274787ea037589b8e4ea8f781a09948294bb0a7
asset="${KOTSADM_BASELINE_CARRIER:?}"
work="$(mktemp -d)"
trap 'rm -rf -- "$work"' EXIT

printf '%s  %s\n' "${archive_sha#sha256:}" "$asset" | sha256sum -c -
python3 - "$asset" "$work" <<'PY'
from pathlib import Path
import sys
import tarfile

with tarfile.open(Path(sys.argv[1]), "r:gz") as archive:
    archive.extractall(Path(sys.argv[2]), filter="data")
PY
grep -Fq 'Kustomize 5.8.1 rendered and applied the expected ConfigMap on baseline x86-64. Guest egress was blocked.' \
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
    select(.files == {"usr/bin/kustomize":{
        "old":"sha256:80780972ee1a89f54612879927f93c14bad9f948f40a1313a065266615418ea2",
        "new":"sha256:5bed1d28563f1525f7fa07c8d0d505f53c408a8b7b84809a8e05eba229ede4bc"}}) |
    . + {candidate_index: .index, candidate_amd64: .amd64}
' "$work/baseline-overlay-receipt.json" > "$work/receipt.json"

sbom=(--companion kotsadm --original-oci-layout "$work/original-amd64")
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
