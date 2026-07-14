#!/usr/bin/env bash
set -euo pipefail

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
embed_dir="$root_dir/cmd/boxui/ui"

cleanup() {
  rm -rf "$embed_dir"
}
trap cleanup EXIT

cd "$root_dir/ui"
npm run build

install -d -m 0700 "$embed_dir"
cp -r dist "$embed_dir/dist"
find "$embed_dir" -type d -exec chmod 0700 {} +
find "$embed_dir" -type f -exec chmod 0600 {} +

cd "$root_dir"
go test -tags embed_ui ./cmd/boxui -run TestEmbeddedUIAssetIntegrity -count=1
