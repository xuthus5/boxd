#!/usr/bin/env bash
set -euo pipefail

# package-desktop.sh 构建并打包 boxd 桌面应用（deb/rpm/AppImage + .desktop + 裸二进制）。
# 用法: ./scripts/package-desktop.sh [VERSION]
# 依赖: GTK4 + WebKitGTK 6.0 开发库, npm, wails3 CLI

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
version="${1:-$(git -C "$root_dir" describe --tags --always --dirty 2>/dev/null || echo dev)}"
export PATH="$PATH:/usr/local/go/bin:$HOME/go/bin"
export GOPROXY="${GOPROXY:-https://goproxy.io,direct}"

cd "$root_dir/desktop"

echo "==> Building frontend (ui/)"
(cd "$root_dir/ui" && npm run build)

echo "==> Preparing desktop/ui/dist"
install -d -m 0700 "$root_dir/desktop/ui"
rm -rf "$root_dir/desktop/ui/dist"
cp -r "$root_dir/ui/dist" "$root_dir/desktop/ui/dist"
find "$root_dir/desktop/ui" -type d -exec chmod 0700 {} +
find "$root_dir/desktop/ui" -type f -exec chmod 0600 {} +

echo "==> Generating Wails bindings"
wails3 generate bindings -d "$root_dir/ui/src/lib/api/bindings" >/dev/null

echo "==> Building desktop binary"
install -d -m 0700 bin
go build \
  -tags "desktop embed_ui with_gvisor with_quic with_dhcp with_wireguard with_utls with_acme with_clash_api" \
  -ldflags "-X github.com/xuthus5/boxd/internal/core.Version=$version" \
  -o bin/boxd-desktop ./

echo "==> Preparing build assets"
mkdir -p build build/linux/appimage build/linux/nfpm
# 应用图标：从项目生成或使用占位。
if [ ! -f build/appicon.png ]; then
  echo "==> No appicon.png found; generating placeholder"
  (command -v convert >/dev/null 2>&1 && convert -size 256x256 xc:#2563eb -fill white -gravity center -pointsize 96 -annotate 0 "B" build/appicon.png) \
    || python3 -c "import zlib,struct; w=256;h=256;raw=b''.join(b'\x00'+b'\x00\x00\xff\x00'*w for _ in range(h)); comp=zlib.compress(raw); data=b'\x89PNG\r\n\x1a\n'+struct.pack('>II',13,0x49484452)+struct.pack('>IIII',w,h,8,2)+comp+struct.pack('>II',0,0x49444154)+b'IEND\xaeB\x60\x82'; open('build/appicon.png','wb').write(data)"
fi

echo "==> Generating .desktop entry"
wails3 generate .desktop \
  -name "boxd-desktop" \
  -exec "boxd-desktop" \
  -icon "boxd-desktop" \
  -comment "sing-box control plane" \
  -categories "Network;Utility;" \
  -outputfile build/linux/boxd-desktop.desktop

echo "==> Packaging deb/rpm"
wails3 tool package -name boxd-desktop -format deb -config build/linux/nfpm/nfpm.yaml -out bin
wails3 tool package -name boxd-desktop -format rpm -config build/linux/nfpm/nfpm.yaml -out bin

echo "==> Packaging AppImage"
cp bin/boxd-desktop build/linux/appimage/boxd-desktop
cp build/appicon.png build/linux/appimage/appicon.png
# 复制为与 .desktop Icon 名一致的副本，规避 wails3 appimage 插件图标名不匹配问题。
cp build/appicon.png build/linux/appimage/boxd-desktop.png
cp build/linux/boxd-desktop.desktop build/linux/appimage/boxd-desktop.desktop
wails3 generate appimage \
  -binary build/linux/appimage/boxd-desktop \
  -icon build/linux/appimage/boxd-desktop.png \
  -desktopfile build/linux/boxd-desktop.desktop \
  -outputdir bin \
  -builddir build/linux/appimage/build || echo "==> AppImage build skipped (linuxdeploy may be missing)"

echo "==> Output artifacts:"
ls -lh bin/boxd-desktop bin/*.deb bin/*.rpm bin/*.AppImage 2>/dev/null || true
