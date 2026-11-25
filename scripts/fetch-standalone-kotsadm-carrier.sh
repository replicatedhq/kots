#!/usr/bin/env bash
set -euo pipefail

[[ "$GITHUB_REPOSITORY" == replicatedhq/kots && "$GITHUB_EVENT_NAME" == workflow_dispatch ]]
target="$1"
archive_sha=sha256:19ace3d27b99c37485478161d274787ea037589b8e4ea8f781a09948294bb0a7
archive_size=495270677
release_id=457485139
asset_id=789378381
asset=kotsadm-latest-baseline-carrier.tar.gz
gh api "repos/$GITHUB_REPOSITORY/releases/$release_id" |
    jq -e --argjson asset_id "$asset_id" --argjson size "$archive_size" --arg sha "$archive_sha" --arg asset "$asset" '
      .draft and .tag_name == "kotsadm-latest-baseline-carrier-20261124" and (.assets | length == 1) and
      any(.assets[]; .id == $asset_id and .name == $asset and .state == "uploaded" and .size == $size and .digest == $sha)
    ' >/dev/null
gh api --header 'Accept: application/octet-stream' \
    "repos/$GITHUB_REPOSITORY/releases/assets/$asset_id" > "$target"
printf '%s  %s\n' "${archive_sha#sha256:}" "$target" | sha256sum -c -
