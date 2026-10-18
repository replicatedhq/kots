#!/usr/bin/env bash

# Fails unless the given file is a Mach-O universal binary with both x86_64 and arm64 slices.

set -euo pipefail

binary="${1:?usage: verify-universal-binary.sh <path-to-binary>}"

if [[ ! -f "$binary" ]]; then
  echo "universal binary not found at $binary" >&2
  exit 1
fi

info="$(file "$binary")"
echo "$info"

if [[ "$info" != *"Mach-O universal binary"* ]]; then
  echo "$binary is not a Mach-O universal binary" >&2
  exit 1
fi

for arch in x86_64 arm64; do
  if [[ "$info" != *"$arch"* ]]; then
    echo "$binary is missing the $arch slice" >&2
    exit 1
  fi
done
