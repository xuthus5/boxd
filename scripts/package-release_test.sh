#!/usr/bin/env bash
set -euo pipefail

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
output_dir=$(mktemp -d)
extract_dir=$(mktemp -d)
trap 'rm -rf "$output_dir" "$extract_dir"' EXIT

version=v0.1.0-test
"$root_dir/scripts/package-release.sh" "$version" "$output_dir"
archive="$output_dir/boxd_${version}_linux_amd64.tar.gz"

sha256sum -c "$archive.sha256"
tar -xzf "$archive" -C "$extract_dir"
for file in boxd boxd.service boxd.env.example LICENSE-APACHE-2.0 LICENSE-GPL-3.0 THIRD_PARTY_NOTICES.md SOURCE-OFFER.txt OPERATIONS.md README.md README.zh-CN.md; do
  test -f "$extract_dir/$file"
done
test "$(stat -c '%a' "$extract_dir/boxd")" = "700"
test "$($extract_dir/boxd --version)" = "$version"
