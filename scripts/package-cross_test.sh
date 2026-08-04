#!/usr/bin/env bash
set -euo pipefail

# package-cross_test.sh 验证 headless 跨平台（Windows/macOS）打包。
root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
output_dir=$(mktemp -d)
extract_dir=$(mktemp -d)
trap 'rm -rf "$output_dir" "$extract_dir"' EXIT

version=v0.1.0-test

# Windows amd64 归档。
"$root_dir/scripts/package-cross.sh" "$version" "$output_dir" windows amd64
archive="$output_dir/boxd_${version}_windows_amd64.tar.gz"
test -f "$archive"
sha256sum -c "$archive.sha256"
tar -xzf "$archive" -C "$extract_dir"
test -f "$extract_dir/boxd.exe"
test -f "$extract_dir/boxd.env.example"
test -f "$extract_dir/LICENSE-APACHE-2.0"
test -f "$extract_dir/LICENSE-GPL-3.0"
test -f "$extract_dir/SOURCE-OFFER.txt"

# macOS arm64 归档。
"$root_dir/scripts/package-cross.sh" "$version" "$output_dir" darwin arm64
darwin_archive="$output_dir/boxd_${version}_darwin_arm64.tar.gz"
test -f "$darwin_archive"
sha256sum -c "$darwin_archive.sha256"

echo "package-cross.sh OK: windows/darwin archives verified"
