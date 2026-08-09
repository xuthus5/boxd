#!/usr/bin/env bash
set -euo pipefail

# generate-desktop-icons.sh 从 web 版 favicon.svg 生成桌面版三端统一图标。
# 产出: desktop/build/appicon.png (Linux), boxd-desktop.ico (Windows), boxd-desktop.icns (macOS)
# 依赖: ImageMagick (magick/convert), wails3

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
src="$root_dir/ui/public/favicon.svg"
out="$root_dir/desktop/build"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

if command -v magick >/dev/null 2>&1; then
  IMG=magick
else
  IMG=convert
fi

echo "==> Rendering favicon.svg to 1024x1024 source"
$IMG -background none -density 300 "$src" -resize 1024x1024 "$tmp/icon-1024.png"

echo "==> Generating Linux appicon.png (512x512)"
$IMG "$tmp/icon-1024.png" -resize 512x512 "$out/appicon.png"

echo "==> Generating Windows .ico and macOS .icns"
(cd "$out" && wails3 generate icons \
  -input "$tmp/icon-1024.png" \
  -windowsfilename boxd-desktop.ico \
  -macfilename boxd-desktop.icns)

echo "==> Generated:"
ls -lh "$out/appicon.png" "$out/boxd-desktop.ico" "$out/boxd-desktop.icns"
