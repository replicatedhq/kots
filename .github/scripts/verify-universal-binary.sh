#!/usr/bin/env bash

# Fails unless the given file is a Mach-O universal binary with both x86_64 and arm64 slices.
# Invoked by the universal_binaries post hook in deploy/.goreleaser.yaml, before GoReleaser publishes the release.

set -euo pipefail

binary="${1:?usage: verify-universal-binary.sh <path-to-binary>}"

fail() {
  if [[ "${GITHUB_ACTIONS:-}" == "true" ]]; then
    echo "::error::$1" >&2
  else
    echo "$1" >&2
  fi
  exit 1
}

[[ -f "$binary" ]] || fail "universal binary not found at $binary"

# -b omits the filename from the summary line; macOS prints extra per-arch lines that repeat the path, so keep only the first
info="$(file -b "$binary" | sed -n 1p)"
echo "$info"

[[ "$info" == *"Mach-O universal binary"* ]] || fail "$binary is not a Mach-O universal binary"

for arch in x86_64 arm64; do
  [[ "$info" == *"$arch"* ]] || fail "$binary is missing the $arch slice"
done
