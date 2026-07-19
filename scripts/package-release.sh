#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "usage: $0 VERSION OUTPUT_DIR [ARCH ...]" >&2
  exit 2
fi

version=$1
output_dir=$2
shift 2
if [[ $# -gt 0 ]]; then
  arches=("$@")
else
  arches=(amd64 arm64)
fi

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
KERNEL_VERSION="${KERNEL_VERSION:-1.13.14}"
BUILD_TAGS="${BUILD_TAGS:-embed_ui with_gvisor with_quic with_dhcp with_wireguard with_utls with_acme with_clash_api}"
stage_root=$(mktemp -d)
cleanup() {
  rm -rf "$stage_root"
  rm -rf "$root_dir/cmd/boxd/ui"
}
trap cleanup EXIT

mkdir -p "$output_dir"
chmod 0700 "$output_dir"

cd "$root_dir"

echo "Building frontend once for multi-arch packages..."
cd ui && npm run build
find dist -type d -exec chmod 0700 {} +
find dist -type f -exec chmod 0600 {} +
cd "$root_dir"

install -d -m 0700 cmd/boxd/ui
rm -rf cmd/boxd/ui/dist
cp -r ui/dist cmd/boxd/ui/dist
find cmd/boxd/ui -type d -exec chmod 0700 {} +
find cmd/boxd/ui -type f -exec chmod 0600 {} +

# Ensure module cache (and sing-box license path) is populated.
go mod download

sing_box_license="$(go env GOMODCACHE)/github.com/sagernet/sing-box@v${KERNEL_VERSION}/LICENSE"
if [[ ! -f "$sing_box_license" ]]; then
  echo "sing-box GPL license not found at $sing_box_license" >&2
  exit 1
fi

if [[ ! -f deploy/boxd.service ]]; then
  echo "deploy/boxd.service is required for release packages" >&2
  exit 1
fi

package_arch() {
  local arch=$1
  local stage_dir="$stage_root/$arch"
  local archive_name="boxd_${version}_linux_${arch}.tar.gz"
  local binary="$stage_dir/boxd"

  mkdir -p "$stage_dir"
  chmod 0700 "$stage_dir"

  echo "Building linux/${arch} binary..."
  GOOS=linux GOARCH="$arch" CGO_ENABLED=0 go build \
    -tags "$BUILD_TAGS" \
    -ldflags "-X github.com/xuthus5/boxd/internal/core.Version=${version} -X github.com/sagernet/sing-box/constant.Version=${KERNEL_VERSION}" \
    -o "$binary" ./cmd/boxd/
  chmod 0700 "$binary"
  install -m 0600 LICENSE "$stage_dir/LICENSE-APACHE-2.0"
  install -m 0600 THIRD_PARTY_NOTICES.md "$stage_dir/THIRD_PARTY_NOTICES.md"
  install -m 0600 README.md "$stage_dir/README.md"
  install -m 0600 README.zh-CN.md "$stage_dir/README.zh-CN.md"
  install -m 0600 docs/operations.md "$stage_dir/OPERATIONS.md"
  install -m 0600 deploy/boxd.service "$stage_dir/boxd.service"
  if [[ -f deploy/boxd.env.example ]]; then
    install -m 0600 deploy/boxd.env.example "$stage_dir/boxd.env.example"
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
  echo "Packed $output_dir/$archive_name"
}

for arch in "${arches[@]}"; do
  case "$arch" in
    amd64|arm64) package_arch "$arch" ;;
    *)
      echo "unsupported arch: $arch (expected amd64 or arm64)" >&2
      exit 2
      ;;
  esac
done

echo "Release packages ready in $output_dir"
