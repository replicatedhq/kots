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

# prints 4 bytes starting at OFFSET as lowercase hex; callers must have bounds-checked OFFSET against the file size,
# so any od failure here is a real read error and is reported as one
read_hex() {
  local out
  out="$(od -An -v -tx1 -j "$1" -N 4 "$binary")" || fail "failed to read $binary at offset $1"
  tr -d ' \n' <<<"$out"
}

[[ -f "$binary" ]] || fail "universal binary not found at $binary"
[[ -r "$binary" ]] || fail "universal binary at $binary is not readable"

size="$(wc -c <"$binary" | tr -d ' ')"
fat_header_size=8
fat_arch_size=20

((size >= 4)) || fail "$binary is not a Mach-O universal binary"
magic="$(read_hex 0)"
[[ "$magic" == "$FAT_MAGIC" ]] || fail "$binary is not a Mach-O universal binary"

((size >= fat_header_size)) || fail "$binary has a truncated universal binary header"
nfat_hex="$(read_hex 4)"
nfat_arch=$((16#$nfat_hex))
((size >= fat_header_size + nfat_arch * fat_arch_size)) || fail "$binary has a truncated universal binary header"

cpu_types=()
for ((i = 0; i < nfat_arch; i++)); do
  cpu_type="$(read_hex $((fat_header_size + i * fat_arch_size)))"
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
