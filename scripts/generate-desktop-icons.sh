#!/usr/bin/env bash
set -euo pipefail

# generate-desktop-icons.sh 生成桌面版三端统一图标与托盘图标。
# 输入: desktop/build/icon-source.png（1024x1024 透明底源图，由 web/品牌 logo 抠底得到）
# 产出: desktop/build/appicon.png (Linux), boxd-desktop.ico (Windows), boxd-desktop.icns (macOS),
#       desktop/tray-icon.png (Linux/macOS 托盘)
# 依赖: ImageMagick (magick/convert), wails3
#
# 更换 logo 时: 用 ImageMagick 抠底生成新 icon-source.png, 例如
#   magick logo.jpg -fuzz 12% -transparent black desktop/build/icon-source.png

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
src="$root_dir/desktop/build/icon-source.png"
out="$root_dir/desktop/build"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

if command -v magick >/dev/null 2>&1; then
  IMG=magick
else
  IMG=convert
fi

if [ ! -f "$src" ]; then
  echo "icon-source.png not found at $src" >&2
  exit 1
fi

echo "==> Generating Linux appicon.png (512x512)"
$IMG "$src" -resize 512x512 "$out/appicon.png"

echo "==> Generating Windows .ico and macOS .icns"
(cd "$out" && wails3 generate icons \
  -input "$src" \
  -windowsfilename boxd-desktop.ico \
  -macfilename boxd-desktop.icns)

echo "==> Generating tray icon (64x64)"
$IMG "$src" -resize 64x64 "$root_dir/desktop/tray-icon.png"

echo "==> Generated:"
ls -lh "$out/appicon.png" "$out/boxd-desktop.ico" "$out/boxd-desktop.icns" "$root_dir/desktop/tray-icon.png"
