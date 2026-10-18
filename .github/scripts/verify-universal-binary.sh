#!/usr/bin/env bash

# Fails unless the given file is a Mach-O universal binary with both x86_64 and arm64 slices.
# Invoked by the universal_binaries post hook in deploy/.goreleaser.yaml, before GoReleaser publishes the release.
#
# Reads the fat header directly (big-endian: magic, nfat_arch, then 20-byte fat_arch entries starting with
# cputype) so the result does not depend on the wording of `file` on the build host.

set -euo pipefail

readonly FAT_MAGIC="cafebabe"
readonly CPU_TYPE_X86_64="01000007"
readonly CPU_TYPE_ARM64="0100000c"

binary="${1:?usage: verify-universal-binary.sh <path-to-binary>}"

fail() {
  if [[ "${GITHUB_ACTIONS:-}" == "true" ]]; then
    echo "::error::$1" >&2
  else
    echo "$1" >&2
  fi
  exit 1
}

# prints N bytes starting at OFFSET as lowercase hex
read_hex() {
  od -An -v -tx1 -j "$1" -N "$2" "$binary" 2>/dev/null | tr -d ' \n' || true
}

[[ -f "$binary" ]] || fail "universal binary not found at $binary"

[[ "$(read_hex 0 4)" == "$FAT_MAGIC" ]] || fail "$binary is not a Mach-O universal binary"

nfat_hex="$(read_hex 4 4)"
[[ ${#nfat_hex} -eq 8 ]] || fail "$binary has a truncated universal binary header"
nfat_arch=$((16#$nfat_hex))
cpu_types=()
for ((i = 0; i < nfat_arch; i++)); do
  cpu_type="$(read_hex $((8 + i * 20)) 4)"
  [[ ${#cpu_type} -eq 8 ]] || fail "$binary has a truncated universal binary header"
  cpu_types+=("$cpu_type")
done
echo "$binary: universal binary with $nfat_arch architecture(s), cputypes: ${cpu_types[*]}"

has_cpu_type() {
  local t
  for t in "${cpu_types[@]}"; do
    [[ "$t" == "$1" ]] && return 0
  done
  return 1
}

has_cpu_type "$CPU_TYPE_X86_64" || fail "$binary is missing the x86_64 slice"
has_cpu_type "$CPU_TYPE_ARM64" || fail "$binary is missing the arm64 slice"
