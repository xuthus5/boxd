#!/usr/bin/env bash
set -euo pipefail

# generate-desktop-icons.sh 生成桌面版三端统一图标与托盘图标。
# 输入: desktop/build/icon-source.svg（左顶部品牌徽标的矢量源）
# 产出: desktop/build/icon-source.png, appicon.png (Linux), boxd-desktop.ico (Windows),
#       boxd-desktop.icns (macOS), desktop/tray-icon.png (Linux/macOS 托盘)
# 依赖: rsvg-convert 或 ImageMagick (magick/convert), wails3

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
src_svg="$root_dir/desktop/build/icon-source.svg"
src_png="$root_dir/desktop/build/icon-source.png"
out="$root_dir/desktop/build"

if [ ! -f "$src_svg" ]; then
  echo "icon-source.svg not found at $src_svg" >&2
  exit 1
fi

render_icon() {
  local size="$1" target="$2"
  if command -v rsvg-convert >/dev/null 2>&1; then
    rsvg-convert -w "$size" -h "$size" "$src_svg" -o "$target"
  elif command -v magick >/dev/null 2>&1; then
    magick -background none "$src_svg" -resize "${size}x${size}" "$target"
  elif command -v convert >/dev/null 2>&1; then
    convert -background none "$src_svg" -resize "${size}x${size}" "$target"
  else
    echo "rsvg-convert or ImageMagick is required" >&2
    exit 1
  fi
}

echo "==> Rendering vector icon source (1024x1024)"
render_icon 1024 "$src_png"

echo "==> Generating Linux appicon.png (512x512)"
render_icon 512 "$out/appicon.png"

echo "==> Generating Windows .ico and macOS .icns"
(cd "$out" && wails3 generate icons \
  -input "$src_png" \
  -windowsfilename boxd-desktop.ico \
  -macfilename boxd-desktop.icns)

echo "==> Generating tray/app icon (256x256)"
render_icon 256 "$root_dir/desktop/tray-icon.png"

echo "==> Generated:"
ls -lh "$src_png" "$out/appicon.png" "$out/boxd-desktop.ico" "$out/boxd-desktop.icns" "$root_dir/desktop/tray-icon.png"
