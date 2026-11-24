#!/usr/bin/env bash
set -euo pipefail

[[ "$GITHUB_REPOSITORY" == replicatedhq/kots && "$GITHUB_EVENT_NAME" == workflow_dispatch ]]
[[ "$LEGACY_COMPANION" == standalone-minio-r0 ]]
[[ -z "$LEGACY_KOTS_VERSION" && -z "$LEGACY_ARCHIVE_DIGEST" && -z "$LEGACY_OVERLAY_RECEIPT" ]]

repo=index.docker.io/kotsadm/minio
tag=0.20251015.172955-r0
original=sha256:1abcfa92c385eec382dc66974ff780006a33ac14af81b2cfb1204fe549fdeab8
original_amd64=sha256:d63b91c5185062fe6e1c2ea60e1465a7ce07b0a17424c087d793c3b84c36a2eb
candidate=sha256:95e9317df05abce5c06646774bd9f4d732f42baddcaaed8940673aeb2af664ac
candidate_amd64=sha256:609181f3a0e7fd407aeba3a32b530c940258ec393ba6d28aa30af052f5857a78
archive_sha=sha256:c9d628eacb1efb05ea3d57978b32926c41d8fc5733b4461d48c2719a929974af
asset="${MINIO_BASELINE_CARRIER:?}"
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
grep -Fq 'its unchanged mc passed a byte-matched S3 object round trip.' \
    "$work/baseline-minio-runtime.log"
grep -Fq 'The original MinIO shell path runs on baseline x86-64.' "$work/baseline-minio-runtime.log"

skopeo inspect --raw "docker://$repo@$original" > "$work/original-index.json"
skopeo inspect --raw "oci:$work/baseline-rebase:candidate" > "$work/candidate-index.json"
printf '%s  %s\n' "${original#sha256:}" "$work/original-index.json" | sha256sum -c -
printf '%s  %s\n' "${candidate#sha256:}" "$work/candidate-index.json" | sha256sum -c -
non_amd64='.manifests | map(select(.platform.os != "linux" or .platform.architecture != "amd64"))'
diff -u <(jq -S "$non_amd64" "$work/original-index.json") <(jq -S "$non_amd64" "$work/candidate-index.json")
skopeo copy --preserve-digests "docker://$repo@$original_amd64" "oci:$work/original-amd64:original"
jq -ce --arg original "$original" --arg original_amd64 "$original_amd64" \
    --arg index "$candidate" --arg amd64 "$candidate_amd64" '
    select(.original_index == $original and .original_amd64 == $original_amd64 and
           .candidate_index == $index and .candidate_amd64 == $amd64) |
    select(.unchanged_runtime_config.User == "1001" and .unchanged_runtime_config.Entrypoint == ["minio"] and
        .application_binaries["usr/bin/minio"].sha256 == "sha256:127ea2fc4c396cc0595152759fc4af3705573f5247797560a2c85802f75f9004" and
        .application_binaries["usr/bin/mc"].sha256 == "sha256:24b08883e916efbd4b5fdf81eaae3d659ae9709e267af5d78af4827d268fd1c7")
' "$work/baseline-rebase-receipt.json" > "$work/receipt.json"

sbom=(--companion minio --original-oci-layout "$work/original-amd64")
if cosign download sbom "$repo@$original" > "$work/original-index.spdx.json" 2> "$work/cosign.log"; then
    sbom+=(--original-index-spdx "$work/original-index.spdx.json")
elif ! grep -Fqx -e 'Error: no SBOM found attached to image index' \
                -e 'Error: no sbom attached to reference' "$work/cosign.log"; then
    cat "$work/cosign.log" >&2
    exit 1
fi
python3 scripts/rebind-legacy-image-sbom.py "$work/receipt.json" "$work/baseline-rebase" "$work/sbom" "${sbom[@]}"
SYFT_CHECK_FOR_APP_UPDATE=false "$SYFT_BIN" scan "oci-dir:$work/baseline-rebase" \
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
    "oci:$work/baseline-rebase:candidate" "docker://$repo@$candidate"
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
