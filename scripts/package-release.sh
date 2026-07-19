#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 VERSION OUTPUT_DIR" >&2
  exit 2
fi

version=$1
output_dir=$2
root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
archive_name="boxd_${version}_linux_amd64.tar.gz"
stage_dir=$(mktemp -d)
trap 'rm -rf "$stage_dir"' EXIT

mkdir -p "$output_dir"
chmod 0700 "$output_dir"

cd "$root_dir"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 VERSION="$version" make build

install -m 0700 bin/boxd "$stage_dir/boxd"
install -m 0600 LICENSE "$stage_dir/LICENSE-APACHE-2.0"
install -m 0600 THIRD_PARTY_NOTICES.md "$stage_dir/THIRD_PARTY_NOTICES.md"
install -m 0600 README.md "$stage_dir/README.md"
install -m 0600 README.zh-CN.md "$stage_dir/README.zh-CN.md"
install -m 0600 docs/operations.md "$stage_dir/OPERATIONS.md"

# Local systemd deployment helpers (always include unit file).
if [[ ! -f deploy/boxd.service ]]; then
  echo "deploy/boxd.service is required for release packages" >&2
  exit 1
fi
install -m 0600 deploy/boxd.service "$stage_dir/boxd.service"

# Env template is optional: package when present. Never ship real secrets.
if [[ -f deploy/boxd.env.example ]]; then
  install -m 0600 deploy/boxd.env.example "$stage_dir/boxd.env.example"
fi

sing_box_license="$(go env GOMODCACHE)/github.com/sagernet/sing-box@v${KERNEL_VERSION:-1.13.14}/LICENSE"
if [[ ! -f "$sing_box_license" ]]; then
  echo "sing-box GPL license not found at $sing_box_license" >&2
  exit 1
fi
install -m 0600 "$sing_box_license" "$stage_dir/LICENSE-GPL-3.0"

printf '%s\n' \
  'Corresponding source for this binary is available from the boxd Git tag:' \
  "$version" \
  '' \
  'The exact dependency versions are recorded in go.mod, go.sum, and' \
  'ui/package-lock.json at that tag. Modified distributors must provide the' \
  'corresponding source and build information required by GPL-3.0.' \
  >"$stage_dir/SOURCE-OFFER.txt"
chmod 0600 "$stage_dir/SOURCE-OFFER.txt"

tar -C "$stage_dir" -czf "$output_dir/$archive_name" .
chmod 0600 "$output_dir/$archive_name"
sha256sum "$output_dir/$archive_name" >"$output_dir/$archive_name.sha256"
chmod 0600 "$output_dir/$archive_name.sha256"
