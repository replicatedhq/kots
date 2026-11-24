#!/usr/bin/env bash
set -euo pipefail

[[ "$GITHUB_REPOSITORY" == replicatedhq/kots && "$GITHUB_EVENT_NAME" == workflow_dispatch ]]
target="$1"
archive_sha=sha256:c9d628eacb1efb05ea3d57978b32926c41d8fc5733b4461d48c2719a929974af
archive_size=166519038
release_id=457399913
asset_id=789065542
asset=minio-20251015-r0-baseline-carrier.tar.gz
gh api "repos/$GITHUB_REPOSITORY/releases/$release_id" |
    jq -e --argjson asset_id "$asset_id" --argjson size "$archive_size" --arg sha "$archive_sha" --arg asset "$asset" '
      .draft and .tag_name == "kots-minio-baseline-carrier-20261124" and (.assets | length == 1) and
      any(.assets[]; .id == $asset_id and .name == $asset and .state == "uploaded" and .size == $size and .digest == $sha)
    ' >/dev/null
gh api --header 'Accept: application/octet-stream' \
    "repos/$GITHUB_REPOSITORY/releases/assets/$asset_id" > "$target"
printf '%s  %s\n' "${archive_sha#sha256:}" "$target" | sha256sum -c -
