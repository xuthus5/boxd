#!/usr/bin/env bash
set -euo pipefail

# build-desktop.sh 构建 boxd-desktop 桌面应用。
# 用法: ./scripts/build-desktop.sh [VERSION]

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
version="${1:-$(git -C "$root_dir" describe --tags --always --dirty 2>/dev/null || echo dev)}"
kernel_version="${KERNEL_VERSION:-1.13.18}"
export PATH="$PATH:/usr/local/go/bin:$HOME/go/bin"
export GOPROXY="${GOPROXY:-https://proxy.golang.org,https://goproxy.io,direct}"
# Wails3 Linux 桌面依赖 GTK4/WebKitGTK（cgo），必须开启 CGO。
export CGO_ENABLED=1
export GOOS=linux

echo "==> Building frontend"
(cd "$root_dir/ui" && npm run build)

echo "==> Copying frontend dist to desktop/ui/dist"
install -d -m 0700 "$root_dir/desktop/ui"
rm -rf "$root_dir/desktop/ui/dist"
cp -r "$root_dir/ui/dist" "$root_dir/desktop/ui/dist"
find "$root_dir/desktop/ui" -type d -exec chmod 0700 {} +
find "$root_dir/desktop/ui" -type f -exec chmod 0600 {} +

echo "==> Generating Wails bindings"
(cd "$root_dir/desktop" && wails3 generate bindings -d "$root_dir/ui/src/lib/api/bindings" >/dev/null)

echo "==> Building desktop binary"
install -d -m 0700 "$root_dir/desktop/bin"
(cd "$root_dir/desktop" && go build \
  -tags "desktop embed_ui with_gvisor with_quic with_dhcp with_wireguard with_utls with_acme with_clash_api" \
  -ldflags "-X github.com/xuthus5/boxd/internal/core.Version=$version -X github.com/sagernet/sing-box/constant.Version=$kernel_version" \
  -o bin/boxd-desktop ./)

echo "==> Built desktop/bin/boxd-desktop"
