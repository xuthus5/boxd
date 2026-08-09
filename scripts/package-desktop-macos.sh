#!/usr/bin/env bash
set -euo pipefail

# package-desktop-macos.sh 构建并打包 boxd 桌面应用的 macOS 版本。
# 用法: ./scripts/package-desktop-macos.sh [VERSION] [ARCH]
#   VERSION  版本号（默认 git describe）
#   ARCH     amd64 或 arm64（默认 arm64；需在对应架构 macOS runner 原生构建）
# 在 macOS runner 上运行。产出: boxd-desktop.app + zip 压缩包。
# DMG 生成依赖 hdiutil（macOS 自带），失败时仅跳过 DMG 不阻断。

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
version="${1:-$(git -C "$root_dir" describe --tags --always --dirty 2>/dev/null || echo dev)}"
arch="${2:-arm64}"
kernel_version="${KERNEL_VERSION:-1.13.14}"
export PATH="$PATH:/usr/local/go/bin:$HOME/go/bin:$(go env GOPATH 2>/dev/null)/bin"
export GOPROXY="${GOPROXY:-https://proxy.golang.org,https://goproxy.io,direct}"
export GOOS=darwin
export CGO_ENABLED=1
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
find "$root_dir/desktop/ui" -type d -exec chmod 0700 {} +
find "$root_dir/desktop/ui" -type f -exec chmod 0600 {} +

echo "==> Generating Wails bindings"
wails3 generate bindings -d "$root_dir/ui/src/lib/api/bindings" >/dev/null

echo "==> Building macOS desktop binary (${arch})"
mkdir -p bin
go build \
  -tags "desktop embed_ui with_gvisor with_quic with_dhcp with_wireguard with_utls with_acme with_clash_api" \
  -ldflags "-X github.com/xuthus5/boxd/internal/core.Version=$version -X github.com/sagernet/sing-box/constant.Version=$kernel_version" \
  -o bin/boxd-desktop ./

echo "==> Assembling .app bundle"
app_dir="build/macos/boxd-desktop.app"
rm -rf "$app_dir"
mkdir -p "$app_dir/Contents/MacOS"
install -m 0755 bin/boxd-desktop "$app_dir/Contents/MacOS/boxd-desktop"
cat >"$app_dir/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key>
  <string>boxd</string>
  <key>CFBundleDisplayName</key>
  <string>boxd</string>
  <key>CFBundleIdentifier</key>
  <string>com.boxd.desktop</string>
  <key>CFBundleVersion</key>
  <string>__VERSION__</string>
  <key>CFBundleShortVersionString</key>
  <string>__VERSION__</string>
  <key>CFBundleExecutable</key>
  <string>boxd-desktop</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>LSMinimumSystemVersion</key>
  <string>11.0</string>
  <key>NSHighResolutionCapable</key>
  <true/>
</dict>
</plist>
PLIST
sed -i.bak "s/__VERSION__/${version}/g" "$app_dir/Contents/Info.plist"
rm -f "$app_dir/Contents/Info.plist.bak"
# 应用图标（若有）。
if [[ -f build/appicon.png ]]; then
  mkdir -p "$app_dir/Contents/Resources"
  install -m 0644 build/appicon.png "$app_dir/Contents/Resources/appicon.png"
fi

echo "==> Packaging zip"
zip_name="boxd-desktop-${version}-macos-${arch}.zip"
(cd build/macos && zip -q -r -9 "$root_dir/$zip_name" boxd-desktop.app)
chmod 0600 "$root_dir/$zip_name"
echo "==> Output: $root_dir/$zip_name"

echo "==> Attempting DMG (best-effort)"
if command -v hdiutil >/dev/null 2>&1; then
  dmg_name="boxd-desktop-${version}-macos-${arch}.dmg"
  hdiutil create -volname boxd -srcfolder "$app_dir" -ov -format UDZO "$root_dir/$dmg_name" >/dev/null 2>&1 \
    && chmod 0600 "$root_dir/$dmg_name" \
    && echo "DMG created: $root_dir/$dmg_name" \
    || echo "DMG creation skipped/failed (non-fatal)"
fi
ls -lh "$root_dir"/*.zip "$root_dir"/*.dmg 2>/dev/null || true
