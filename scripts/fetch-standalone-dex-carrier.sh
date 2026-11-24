#!/usr/bin/env bash
set -euo pipefail

[[ "$GITHUB_REPOSITORY" == replicatedhq/kots && "$GITHUB_EVENT_NAME" == workflow_dispatch ]]
target="$1"
archive_sha=sha256:f93c6373dde0d671710e7fb9d1cf3d890ffc826d8caf2a410e108d00666f8d69
archive_size=111618904
release_id=457394462
asset_id=789046583
asset=dex-2.44.0-baseline-carrier.tar.gz
gh api "repos/$GITHUB_REPOSITORY/releases/$release_id" |
    jq -e --argjson asset_id "$asset_id" --argjson size "$archive_size" --arg sha "$archive_sha" --arg asset "$asset" '
      .draft and .tag_name == "kots-dex-baseline-carrier-20261124" and (.assets | length == 1) and
      any(.assets[]; .id == $asset_id and .name == $asset and .state == "uploaded" and .size == $size and .digest == $sha)
    ' >/dev/null
gh api --header 'Accept: application/octet-stream' \
    "repos/$GITHUB_REPOSITORY/releases/assets/$asset_id" > "$target"
printf '%s  %s\n' "${archive_sha#sha256:}" "$target" | sha256sum -c -
