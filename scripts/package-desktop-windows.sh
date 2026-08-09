#!/usr/bin/env bash
set -euo pipefail

# package-desktop-windows.sh 构建并打包 boxd 桌面应用的 Windows 版本。
# 用法: ./scripts/package-desktop-windows.sh [VERSION] [ARCH]
#   VERSION  版本号（默认 git describe）
#   ARCH     amd64 或 arm64（默认 amd64）
# 在 windows-2022 runner 上运行（自带 git-bash）。
# 产出: boxd-desktop.exe + boxd-desktop-<version>-windows-<arch>.zip

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
version="${1:-$(git -C "$root_dir" describe --tags --always --dirty 2>/dev/null || echo dev)}"
arch="${2:-amd64}"
kernel_version="${KERNEL_VERSION:-1.13.14}"
export PATH="$PATH:/usr/local/go/bin:$HOME/go/bin:/c/Go/bin:$(go env GOPATH 2>/dev/null)/bin"
export GOPROXY="${GOPROXY:-https://proxy.golang.org,https://goproxy.io,direct}"
export GOOS=windows
export CGO_ENABLED=0
export GOARCH="$arch"

cd "$root_dir/desktop"

echo "==> Building frontend (ui/)"
if [[ ! -d "$root_dir/ui/node_modules" ]]; then
  (cd "$root_dir/ui" && npm ci)
fi
(cd "$root_dir/ui" && npm run build)

echo "==> Preparing desktop/ui/dist"
rm -rf "$root_dir/desktop/ui"
mkdir -p "$root_dir/desktop/ui"
cp -r "$root_dir/ui/dist" "$root_dir/desktop/ui/dist"

echo "==> Generating Wails bindings"
wails3 generate bindings -d "$root_dir/ui/src/lib/api/bindings" >/dev/null

echo "==> Building Windows desktop binary (${arch})"
mkdir -p bin
go build \
  -tags "desktop embed_ui with_gvisor with_quic with_dhcp with_wireguard with_utls with_acme with_clash_api" \
  -ldflags "-X github.com/xuthus5/boxd/internal/core.Version=$version -X github.com/sagernet/sing-box/constant.Version=$kernel_version" \
  -o bin/boxd-desktop.exe ./

echo "==> Packaging zip"
zip_name="boxd-desktop-${version}-windows-${arch}.zip"
make_zip() {
  local src="$1" dest="$2"
  if command -v zip >/dev/null 2>&1; then
    (cd "$src" && zip -q -9 "$dest" boxd-desktop.exe)
  elif command -v 7z >/dev/null 2>&1; then
    (cd "$src" && 7z a -tzip "$dest" boxd-desktop.exe >/dev/null)
  elif [ -f "/c/Program Files/7-Zip/7z.exe" ]; then
    (cd "$src" && "/c/Program Files/7-Zip/7z.exe" a -tzip "$dest" boxd-desktop.exe >/dev/null)
  else
    echo "zip: no implementation found (need zip or 7z)" >&2
    return 1
  fi
}
make_zip bin "$root_dir/$zip_name"
chmod 0600 "$root_dir/$zip_name"
echo "==> Output: $root_dir/$zip_name"
ls -lh "$root_dir/$zip_name" bin/boxd-desktop.exe
