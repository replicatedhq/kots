#!/usr/bin/env bash
set -euo pipefail

: "${LEGACY_KOTS_VERSION:?}"
: "${LEGACY_ARCHIVE_DIGEST:?}"
: "${LEGACY_OVERLAY_RECEIPT:?}"

if [[ "$GITHUB_REPOSITORY" != replicatedhq/kots || "$GITHUB_EVENT_NAME" != workflow_dispatch ]]; then
  echo "Run this job through the KOTS repository's manual workflow."
  exit 1
fi
if [[ ! "$LEGACY_KOTS_VERSION" =~ ^v1\.(10[89]|11[0-9]|12[0-9])\.(0|[1-9][0-9]?)$ ]]; then
  echo "Use a stable KOTS version from v1.108.4 through v1.129.3."
  exit 1
fi
minor="${BASH_REMATCH[1]}"
patch="${BASH_REMATCH[2]}"
if (( (minor == 108 && patch < 4) || (minor == 129 && patch > 3) )); then
  echo "The KOTS version is outside the legacy backfill."
  exit 1
fi

companion="${LEGACY_COMPANION:-}"
repo=index.docker.io/kotsadm/kotsadm
source=127.0.0.1:5077/kotsadm
image_tag="$LEGACY_KOTS_VERSION"
asset=kotsadm-nominio.tar.gz
if [[ -n "$companion" ]]; then
  if [[ "$LEGACY_KOTS_VERSION" != v1.108.4 ]]; then
    echo "Companion publication currently covers v1.108.4."
    exit 1
  fi
  repo="index.docker.io/kotsadm/$companion"
  source="127.0.0.1:5077/$companion"
  archive_pin=sha256:1dd98ee58b35f501a480916f6d3bfe7af865edb494f479c726f3fc7c3fc2cfe4
  case "$companion" in
    dex)
      image_tag=2.39.0-r0
      original_pin=sha256:bdce436cd795c560abe3e2722eaff6e12617eb36eff6018a1963a45e7192c077
      candidate_pin=sha256:f1f61ba8a8e95d43963b4808d91d7616d4c1f0fd88d707edfd1c0cf026e1a9db
      ;;
    rqlite)
      image_tag=8.23.0-r0
      original_pin=sha256:bf87ad111537d8489097e6fcd849afb3617d792a70de1018b8fda57b272753ae
      candidate_pin=sha256:9b25cdfbf58603d009bc3e9c4fb40a438d1cfd503000b518bd964aa1bd3bf2c0
      ;;
    minio)
      image_tag=0.20231220.010002-r4
      asset=kotsadm.tar.gz
      archive_pin=sha256:4bd26307ce4db5708c6c6931c7601833dbfda5b28bd0062c44c1dbcb94621179
      original_pin=sha256:188cb0331320fe417024f9ec63f95e75e3b33b8e7f680370ef7dd3bbbc97e514
      candidate_pin=sha256:0a937784801bf7d51aef0b7d1f507c39aff9a6fa531592b46e9483f7a6cab147
      ;;
    local-volume-provider)
      repo=index.docker.io/replicated/local-volume-provider
      image_tag=v0.6.1
      original_pin=sha256:a4241a6cf3d50e1f3529f3af16c7b1af295c0d26f82bc41475906b021f39ada7
      candidate_pin=sha256:6ee35c7a77d8731bfee416a40e6c09e897affc0e350da01e85688232c318e7b2
      ;;
    *)
      echo "Use dex, rqlite, minio, or local-volume-provider for the v1.108.4 companion."
      exit 1
      ;;
  esac
  LEGACY_OVERLAY_RECEIPT="$(jq -ce '
    select(type == "object" and (has("baseline_userland_index") | not)) |
    . + {candidate_index: .index, candidate_amd64: .amd64}
  ' <<<"$LEGACY_OVERLAY_RECEIPT")"
fi
LEGACY_ORIGINAL_INDEX="$(jq -er '.original_index' <<<"$LEGACY_OVERLAY_RECEIPT")"
LEGACY_CANDIDATE_INDEX="$(jq -er '.candidate_index' <<<"$LEGACY_OVERLAY_RECEIPT")"
LEGACY_CANDIDATE_AMD64="$(jq -er '.candidate_amd64' <<<"$LEGACY_OVERLAY_RECEIPT")"
baseline_parent="$(jq -r '.baseline_userland_index // empty' <<<"$LEGACY_OVERLAY_RECEIPT")"
if [[ -n "$companion" ]] && [[ "$LEGACY_ARCHIVE_DIGEST" != "$archive_pin" ||
    "$LEGACY_ORIGINAL_INDEX" != "$original_pin" || "$LEGACY_CANDIDATE_INDEX" != "$candidate_pin" ]]; then
  echo "The companion receipt and archive must match the tested v1.108.4 digests."
  exit 1
fi
for digest in "$LEGACY_ARCHIVE_DIGEST" "$LEGACY_ORIGINAL_INDEX" "$LEGACY_CANDIDATE_INDEX" "$LEGACY_CANDIDATE_AMD64"; do
  if [[ ! "$digest" =~ ^sha256:[0-9a-f]{64}$ ]]; then
    echo "Each input digest must be a SHA-256 digest."
    exit 1
  fi
done

work="$(mktemp -d)"
container=""
cleanup() {
  if [[ -n "$container" ]]; then docker rm -f "$container" >/dev/null || true; fi
  rm -rf -- "$work"
}
trap cleanup EXIT

release="$(gh api "repos/$GITHUB_REPOSITORY/releases/tags/$LEGACY_KOTS_VERSION")"
jq -e --arg asset "$asset" --arg digest "$LEGACY_ARCHIVE_DIGEST" '
  any(.assets[]; .name == $asset and .state == "uploaded" and .digest == $digest)
' <<<"$release" >/dev/null
gh release download "$LEGACY_KOTS_VERSION" -R "$GITHUB_REPOSITORY" -p "$asset" -D "$work"
printf '%s  %s\n' "${LEGACY_ARCHIVE_DIGEST#sha256:}" "$work/$asset" | sha256sum -c -
tar -xzf "$work/$asset" -C "$work"
container="$(docker run -d --rm -p 127.0.0.1:5077:5000 \
  --mount "type=bind,source=$work/images,target=/var/lib/registry,readonly" \
  docker.io/library/registry@sha256:a3d8aaa63ed8681a604f1dea0aa03f100d5895b6a58ace528858a7b332415373)"
for attempt in {1..20}; do
  if curl -fsS http://127.0.0.1:5077/v2/ >/dev/null 2>&1; then break; fi
  if [[ "$attempt" == 20 ]]; then docker logs "$container"; exit 1; fi
  sleep 1
done

skopeo inspect --raw --tls-verify=false "docker://$source:$image_tag" > "$work/candidate-index.json"
skopeo inspect --raw "docker://$repo@$LEGACY_ORIGINAL_INDEX" > "$work/original-index.json"
check_digest() { printf '%s  %s\n' "${2#sha256:}" "$1" | sha256sum -c -; }
check_digest "$work/candidate-index.json" "$LEGACY_CANDIDATE_INDEX"
check_digest "$work/original-index.json" "$LEGACY_ORIGINAL_INDEX"

amd64_filter='.manifests[] | select(.platform.os == "linux" and .platform.architecture == "amd64") | .digest'
old_amd64="$(jq -er "$amd64_filter" "$work/original-index.json")"
new_amd64="$(jq -er "$amd64_filter" "$work/candidate-index.json")"
test "$old_amd64" != "$new_amd64"
test "$new_amd64" = "$LEGACY_CANDIDATE_AMD64"
if [[ -n "$companion" ]]; then
  test "$old_amd64" = "$(jq -er '.original_amd64' <<<"$LEGACY_OVERLAY_RECEIPT")"
fi
non_amd64='.manifests | map(select(.platform.os != "linux" or .platform.architecture != "amd64"))'
diff -u <(jq -S "$non_amd64" "$work/original-index.json") <(jq -S "$non_amd64" "$work/candidate-index.json")

skopeo inspect --raw --tls-verify=false "docker://$source@$new_amd64" > "$work/candidate-amd64.json"
skopeo inspect --raw "docker://$repo@$old_amd64" > "$work/original-amd64.json"
check_digest "$work/candidate-amd64.json" "$new_amd64"
check_digest "$work/original-amd64.json" "$old_amd64"
skopeo inspect --raw --config --tls-verify=false "docker://$source@$new_amd64" > "$work/candidate-config.json"
skopeo inspect --raw --config "docker://$repo@$old_amd64" > "$work/original-config.json"
check_digest "$work/candidate-config.json" "$(jq -er '.config.digest' "$work/candidate-amd64.json")"
check_digest "$work/original-config.json" "$(jq -er '.config.digest' "$work/original-amd64.json")"
if [[ -n "$baseline_parent" ]]; then
  test "$baseline_parent" = sha256:81323e5d194abc9bb3c9b7b50a1449bda1315eaa1021f3015218b62982d7813a
  jq -eS 'del(.rootfs, .history)' "$work/original-config.json" > "$work/original-runtime.json"
  jq -eS 'del(.rootfs, .history)' "$work/candidate-config.json" > "$work/candidate-runtime.json"
else
  diff -u <(jq -S '.layers' "$work/original-amd64.json") <(jq -S '.layers[:-1]' "$work/candidate-amd64.json")
  jq -eS '.history |= (. // [])' "$work/original-config.json" > "$work/original-runtime.json"
  jq -eS '.history |= .[:-1] | .rootfs.diff_ids |= .[:-1]' "$work/candidate-config.json" > "$work/candidate-runtime.json"
fi
diff -u "$work/original-runtime.json" "$work/candidate-runtime.json"

current_digest() {
  skopeo inspect --raw "$@" "docker://$repo:$image_tag" | sha256sum | awk '{print "sha256:"$1}'
}
assert_public_predecessor() {
  live="$(current_digest "$@")"
  if [[ "$live" != "$LEGACY_ORIGINAL_INDEX" && "$live" != "$LEGACY_CANDIDATE_INDEX" ]]; then
    echo "The current Docker Hub tag differs from both verified digests: $live"
    exit 1
  fi
}
assert_public_predecessor
printf '%s\n' "$LEGACY_OVERLAY_RECEIPT" > "$work/overlay-receipt.json"
fetch_sbom() {
  local subject="$1" arch="$2"
  local output="$work/original-$arch.spdx.json"
  local errors="$work/cosign-$arch.log"
  if cosign download sbom "$repo@$subject" > "$output" 2> "$errors"; then
    test -s "$output"
    sbom_args+=("--original-$arch-spdx" "$output")
  elif grep -Fqx -e 'Error: no SBOM found attached to image index' -e 'Error: no sbom attached to reference' "$errors"; then
    echo "Original $arch image has no legacy SBOM attachment."
  else
    cat "$errors" >&2
    return 1
  fi
}
sbom_args=()
fetch_sbom "$LEGACY_ORIGINAL_INDEX" index
if [[ -n "$baseline_parent" || -n "$companion" ]]; then
  skopeo copy --preserve-digests "docker://$repo@$old_amd64" "oci:$work/original-amd64:original"
  sbom_args+=(--original-oci-layout "$work/original-amd64")
else
  fetch_sbom "$old_amd64" amd64
fi
if [[ -n "$companion" ]]; then
  sbom_args+=(--companion "$companion")
fi
skopeo copy --all --preserve-digests --src-tls-verify=false \
  "docker://$source@$LEGACY_CANDIDATE_INDEX" "oci:$work/oci:candidate"
python3 scripts/rebind-legacy-image-sbom.py "$work/overlay-receipt.json" "$work/oci" "$work/sbom" "${sbom_args[@]}"
if [[ -n "$baseline_parent" || -n "$companion" ]]; then
  : "${SYFT_BIN:?}"
  SYFT_CHECK_FOR_APP_UPDATE=false "$SYFT_BIN" scan "oci-dir:$work/oci" --platform linux/amd64 -o spdx-json > "$work/sbom/amd64.spdx.json"
  jq -e '.spdxVersion == "SPDX-2.3" and (.packages | length > 0)' "$work/sbom/amd64.spdx.json" >/dev/null
fi
echo "Verified $repo:$image_tag, its original arm64 descriptor, and any historical SBOM attachments."
if [[ "${LEGACY_PUBLISH:-false}" != true ]]; then exit 0; fi

: "${DOCKERHUB_USER:?}"
: "${DOCKERHUB_PASSWORD:?}"
export DOCKER_CONFIG="$work/auth"
install -d -m 700 "$DOCKER_CONFIG"
printf '%s' "$DOCKERHUB_PASSWORD" | docker login --username "$DOCKERHUB_USER" --password-stdin
unset DOCKERHUB_PASSWORD
auth="$DOCKER_CONFIG/config.json"

# Stage the immutable digest and its SBOM before moving the release tag.
skopeo copy --all --preserve-digests --src-tls-verify=false --dest-authfile "$auth" \
  "docker://$source@$LEGACY_CANDIDATE_INDEX" "docker://$repo@$LEGACY_CANDIDATE_INDEX"
for binding in "$new_amd64:amd64" "$LEGACY_CANDIDATE_INDEX:index"; do
  digest="${binding%:*}"
  output="$work/sbom/${binding##*:}.spdx.json"
  if [[ -f "$output" ]]; then
    cosign attach sbom --type spdx --sbom "$output" "$repo@$digest"
    cosign download sbom "$repo@$digest" | jq -cS . | cmp - <(jq -cS . "$output")
  fi
done

assert_public_predecessor --authfile "$auth"
if [[ "$live" == "$LEGACY_ORIGINAL_INDEX" ]]; then
  skopeo copy --all --preserve-digests --authfile "$auth" \
    "docker://$repo@$LEGACY_CANDIDATE_INDEX" "docker://$repo:$image_tag"
fi
for attempt in {1..10}; do
  live="$(current_digest --authfile "$auth")"
  if [[ "$live" == "$LEGACY_CANDIDATE_INDEX" ]]; then
    echo "Verified $repo:$image_tag at $live."
    exit 0
  fi
  sleep 3
done
echo "Docker Hub did not return the published index."
exit 1
