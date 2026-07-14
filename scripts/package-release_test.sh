#!/usr/bin/env bash
set -euo pipefail

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
output_dir=$(mktemp -d)
extract_dir=$(mktemp -d)
trap 'rm -rf "$output_dir" "$extract_dir"' EXIT

version=v0.1.0-test
"$root_dir/scripts/package-release.sh" "$version" "$output_dir"
archive="$output_dir/boxui_${version}_linux_amd64.tar.gz"

sha256sum -c "$archive.sha256"
tar -xzf "$archive" -C "$extract_dir"
for file in boxui LICENSE-MIT LICENSE-GPL-3.0 THIRD_PARTY_NOTICES.md SOURCE-OFFER.txt OPERATIONS.md; do
  test -f "$extract_dir/$file"
done
test "$(stat -c '%a' "$extract_dir/boxui")" = "700"
test "$($extract_dir/boxui --version)" = "$version"
