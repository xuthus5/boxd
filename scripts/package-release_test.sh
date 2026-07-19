#!/usr/bin/env bash
set -euo pipefail

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
output_dir=$(mktemp -d)
extract_amd64=$(mktemp -d)
extract_arm64=$(mktemp -d)
trap 'rm -rf "$output_dir" "$extract_amd64" "$extract_arm64"' EXIT

version=v0.1.0-test
"$root_dir/scripts/package-release.sh" "$version" "$output_dir"

for arch in amd64 arm64; do
  archive="$output_dir/boxd_${version}_linux_${arch}.tar.gz"
  test -f "$archive"
  sha256sum -c "$archive.sha256"
done

tar -xzf "$output_dir/boxd_${version}_linux_amd64.tar.gz" -C "$extract_amd64"
tar -xzf "$output_dir/boxd_${version}_linux_arm64.tar.gz" -C "$extract_arm64"

for extract_dir in "$extract_amd64" "$extract_arm64"; do
  for file in boxd boxd.service boxd.env.example LICENSE-APACHE-2.0 LICENSE-GPL-3.0 THIRD_PARTY_NOTICES.md SOURCE-OFFER.txt OPERATIONS.md README.md README.zh-CN.md; do
    test -f "$extract_dir/$file"
  done
  test "$(stat -c '%a' "$extract_dir/boxd")" = "700"
done

host_arch=$(uname -m)
case "$host_arch" in
  x86_64|amd64) test "$("$extract_amd64/boxd" --version)" = "$version" ;;
  aarch64|arm64) test "$("$extract_arm64/boxd" --version)" = "$version" ;;
esac

amd64_info=$(file "$extract_amd64/boxd")
arm64_info=$(file "$extract_arm64/boxd")
printf '%s
' "$amd64_info" | grep -Eiq 'x86-64|x86_64'
printf '%s
' "$arm64_info" | grep -Eiq 'aarch64|ARM aarch64|ARM64'
