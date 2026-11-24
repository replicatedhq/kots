#!/usr/bin/env bash
set -euo pipefail

[[ "${GITHUB_REPOSITORY,,}" == replicatedhq/kots && "$GITHUB_EVENT_NAME" == workflow_dispatch ]]
case "${BACKFILL_PHASE:-}" in
  stage|finalize|restore-origin-pointer) ;;
  *) echo 'Specify stage, finalize, or restore-origin-pointer.' >&2; exit 1 ;;
esac
[[ "${BACKFILL_DRAFT_ID:-}" =~ ^[1-9][0-9]*$ ]]
[[ "${BACKFILL_PLAN_ASSET_ID:-}" =~ ^[1-9][0-9]*$ ]]
[[ "${BACKFILL_PLAN_SHA256:-}" =~ ^[a-f0-9]{64}$ ]]
if [[ "$BACKFILL_PHASE" == stage ]]; then
  [[ "${BACKFILL_ASSET_ID:-}" =~ ^[1-9][0-9]*$ ]]
  [[ "$BACKFILL_ASSET_ID" != "$BACKFILL_PLAN_ASSET_ID" ]]
else
  [[ -z "${BACKFILL_ASSET_ID:-}" ]]
fi

work="$RUNNER_TEMP/legacy-kurl-origin"
plan="$work/plan.json"
publisher=scripts/publish-legacy-kurl.py

checksum() {
  printf '%s  %s\n' "$1" "$2" | sha256sum --check --strict
}

asset_meta() {
  jq -e --arg id "$1" --arg digest "sha256:$2" '
    .assets[] | select((.id|tostring) == $id and .state == "uploaded" and .digest == $digest and .size > 0)
  ' "$work/draft.json"
}

check_plan() {
  checksum "$BACKFILL_PLAN_SHA256" "$plan"
  python3 "$publisher" check-plan --plan "$plan"
  local version
  version="$(jq -er '.version' "$plan")"
  [[ "$version" == "${ADDON_VERSION#v}" ]]
}

case "${1:-}" in
  verify)
    mkdir -m 700 "$work"
    gh api "repos/replicatedhq/kots/releases/$BACKFILL_DRAFT_ID" > "$work/draft.json"
    jq -e --arg id "$BACKFILL_DRAFT_ID" '.draft and (.id|tostring) == $id' "$work/draft.json" >/dev/null
    asset_meta "$BACKFILL_PLAN_ASSET_ID" "$BACKFILL_PLAN_SHA256" > "$work/plan-asset.json"
    gh api -H 'Accept: application/octet-stream' "repos/replicatedhq/kots/releases/assets/$BACKFILL_PLAN_ASSET_ID" > "$plan"
    [[ "$(wc -c < "$plan")" -eq "$(jq -er '.size' "$work/plan-asset.json")" ]]
    check_plan
    if [[ "$BACKFILL_PHASE" == stage ]]; then
      candidate="$work/candidate.tar.gz"
      candidate_sha="$(jq -er '.candidate.sha256' "$plan")"
      candidate_size="$(jq -er '.candidate.size' "$plan")"
      asset_meta "$BACKFILL_ASSET_ID" "$candidate_sha" > "$work/candidate-asset.json"
      [[ "$candidate_size" -eq "$(jq -er '.size' "$work/candidate-asset.json")" ]]
      gh api -H 'Accept: application/octet-stream' "repos/replicatedhq/kots/releases/assets/$BACKFILL_ASSET_ID" > "$candidate"
      [[ "$(wc -c < "$candidate")" -eq "$candidate_size" ]]
      checksum "$candidate_sha" "$candidate"
    fi
    ;;
  apply)
    check_plan
    if [[ "$BACKFILL_PHASE" == stage ]]; then
      python3 "$publisher" stage --plan "$plan" --candidate "$work/candidate.tar.gz"
    else
      python3 "$publisher" "$BACKFILL_PHASE" --plan "$plan"
    fi
    ;;
  *)
    echo 'Specify verify or apply.' >&2
    exit 1
    ;;
esac
