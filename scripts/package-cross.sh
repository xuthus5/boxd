#!/usr/bin/env bash
set -euo pipefail

# package-cross.sh 构建并打包 headless boxd 的 Windows/macOS 发布归档。
# 用法: ./scripts/package-cross.sh VERSION OUTPUT_DIR OS [ARCH ...]
#   VERSION   版本号（如 nightly 或 v0.1.0）
#   OUTPUT_DIR 输出目录
#   OS        windows 或 darwin
#   ARCH      amd64 / arm64（默认 amd64 arm64）
# 依赖: Go, npm（构建前端一次）

if [[ $# -lt 3 ]]; then
  echo "usage: $0 VERSION OUTPUT_DIR OS [ARCH ...]" >&2
  exit 2
fi

version=$1
output_dir=$2
target_os=$3
shift 3
if [[ $# -gt 0 ]]; then
  arches=("$@")
else
  arches=(amd64 arm64)
fi

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
export PATH="$PATH:/usr/local/go/bin:$HOME/go/bin"
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

echo "Building frontend once for cross-platform packages..."
if [[ ! -d "$root_dir/ui/node_modules" ]]; then
  (cd "$root_dir/ui" && npm ci)
fi
cd ui && npm run build
if [[ "$target_os" != "windows" ]]; then
  find dist -type d -exec chmod 0700 {} +
  find dist -type f -exec chmod 0600 {} +
fi
cd "$root_dir"

rm -rf cmd/boxd/ui
mkdir -p cmd/boxd/ui
cp -r ui/dist cmd/boxd/ui/dist
if [[ "$target_os" != "windows" ]]; then
  find cmd/boxd/ui -type d -exec chmod 0700 {} +
  find cmd/boxd/ui -type f -exec chmod 0600 {} +
fi

go mod download

sing_box_license="$(go env GOMODCACHE)/github.com/sagernet/sing-box@v${KERNEL_VERSION}/LICENSE"
if [[ ! -f "$sing_box_license" ]]; then
  echo "sing-box GPL license not found at $sing_box_license" >&2
  exit 1
fi

package_arch() {
  local arch=$1
  local stage_dir="$stage_root/$arch"
  local archive_name="boxd_${version}_${target_os}_${arch}.tar.gz"
  local binary="$stage_dir/boxd"
  [[ "$target_os" == "windows" ]] && binary="$stage_dir/boxd.exe"

  mkdir -p "$stage_dir"
  chmod 0700 "$stage_dir"

  echo "Building ${target_os}/${arch} binary..."
  GOOS="$target_os" GOARCH="$arch" CGO_ENABLED=0 go build \
    -tags "$BUILD_TAGS" \
    -ldflags "-X github.com/xuthus5/boxd/internal/core.Version=${version} -X github.com/sagernet/sing-box/constant.Version=${KERNEL_VERSION}" \
    -o "$binary" ./cmd/boxd/
  chmod 0700 "$binary"
  stage_file LICENSE "$stage_dir/LICENSE-APACHE-2.0"
  stage_file THIRD_PARTY_NOTICES.md "$stage_dir/THIRD_PARTY_NOTICES.md"
  stage_file README.md "$stage_dir/README.md"
  stage_file README.zh-CN.md "$stage_dir/README.zh-CN.md"
  stage_file docs/operations.md "$stage_dir/OPERATIONS.md"
  if [[ -f deploy/boxd.env.example ]]; then
    stage_file deploy/boxd.env.example "$stage_dir/boxd.env.example"
  fi
  stage_file "$sing_box_license" "$stage_dir/LICENSE-GPL-3.0"

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
  sha256sum_impl "$output_dir/$archive_name" >"$output_dir/$archive_name.sha256"
  chmod 0600 "$output_dir/$archive_name.sha256"
  echo "Packed $output_dir/$archive_name"
}

# stage_file 复制文件到归档暂存目录（0600）。
# Windows git-bash 的 install -m 对已存在目标可能报权限错，改用 cp + 容错 chmod。
stage_file() {
  local src="$1" dst="$2"
  cp "$src" "$dst"
  chmod 0600 "$dst" 2>/dev/null || true
}

# sha256sum_impl 计算 sha256（macOS 无 sha256sum，用 shasum -a 256）。
sha256sum_impl() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$@"
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$@"
  else
    echo "sha256sum: no implementation found" >&2
    exit 1
  fi
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

echo "Cross-platform packages ready in $output_dir"
