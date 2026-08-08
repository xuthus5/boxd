#!/usr/bin/env bash
set -euo pipefail

# package-linux.sh 构建并打包 headless（Web 服务形态）boxd 的 deb/rpm/AppImage。
# 用法: ./scripts/package-linux.sh [VERSION] [ARCH ...]
# 依赖: Go, npm, wails3 CLI（内置 nfpm）

if [[ $# -lt 1 ]]; then
  echo "usage: $0 VERSION [ARCH ...]" >&2
  exit 2
fi

version=$1
shift 1
if [[ $# -gt 0 ]]; then
  arches=("$@")
else
  arches=(amd64)
fi

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
source "$root_dir/scripts/lib-version.sh"
pkg_version=$(resolve_package_version "$version")
KERNEL_VERSION="${KERNEL_VERSION:-1.13.14}"
BUILD_TAGS="${BUILD_TAGS:-embed_ui with_gvisor with_quic with_dhcp with_wireguard with_utls with_acme with_clash_api}"
export PATH="$PATH:/usr/local/go/bin:$HOME/go/bin"
export GOPROXY="${GOPROXY:-https://proxy.golang.org,https://goproxy.io,direct}"

cd "$root_dir"

echo "Building frontend once for all packages..."
(cd ui && npm run build)
find ui/dist -type d -exec chmod 0700 {} +
find ui/dist -type f -exec chmod 0600 {} +

install -d -m 0700 cmd/boxd/ui
rm -rf cmd/boxd/ui/dist
cp -r ui/dist cmd/boxd/ui/dist
find cmd/boxd/ui -type d -exec chmod 0700 {} +
find cmd/boxd/ui -type f -exec chmod 0600 {} +
cleanup() {
  rm -rf "$root_dir/cmd/boxd/ui" "$root_dir/desktop/ui"
}
trap cleanup EXIT

go mod download

package_arch() {
  local arch=$1
  local binary="/tmp/boxd-${arch}"
  local out="$root_dir/build/linux/bin"

  install -d -m 0700 "$out"
  chmod 0700 "$out"

  echo "Building linux/${arch} binary..."
  GOOS=linux GOARCH="$arch" CGO_ENABLED=0 go build \
    -tags "$BUILD_TAGS" \
    -ldflags "-X github.com/xuthus5/boxd/internal/core.Version=${pkg_version} -X github.com/sagernet/sing-box/constant.Version=${KERNEL_VERSION}" \
    -o "$binary" ./cmd/boxd/
  chmod 0700 "$binary"

  # 复制到 nfpm 期望的位置（架构化子目录，避免多架构冲突）。
  install -d -m 0700 "$out/$arch"
  cp "$binary" "$out/$arch/boxd"
  chmod 0700 "$out/$arch/boxd"

  # 生成架构化 nfpm 配置（替换 ${GOARCH} 与 ${VERSION}）。
  local nfpm_cfg="$out/$arch/nfpm.yaml"
  sed -e "s/\${GOARCH}/$arch/g" -e "s/version: \"0.0.0\"/version: \"${pkg_version}\"/" \
    "$root_dir/build/linux/nfpm/nfpm.yaml" >"$nfpm_cfg"
  # nfpm contents 的 src 路径相对配置文件，需指向构建产物。
  sed -i -e "s|src: \"./bin/boxd\"|src: \"$out/$arch/boxd\"|" "$nfpm_cfg"

  echo "Packaging linux/${arch} deb and rpm..."
  wails3 tool package -name boxd -format deb -config "$nfpm_cfg" -out "$out"
  wails3 tool package -name boxd -format rpm -config "$nfpm_cfg" -out "$out"
  mv -f "$out/boxd.deb" "$out/boxd_${version}_linux_${arch}.deb" 2>/dev/null || true
  mv -f "$out/boxd.rpm" "$out/boxd_${version}_linux_${arch}.rpm" 2>/dev/null || true

  # AppImage 封装格式与内部二进制架构无关：x86_64 宿主 appimagetool 可封装 arm64 内容。
  echo "Packaging linux/${arch} AppImage..."
  package_appimage "$arch" "$binary" "$out"
  echo "Packed linux/${arch} deb/rpm/AppImage"
}

# appimagetool_path 返回 x86_64 宿主 appimagetool（AppImage 封装格式可承载任意内部架构）。
appimagetool_path() {
  local tool="/tmp/appimagetool"
  if [[ ! -x "$tool" ]]; then
    echo "Downloading appimagetool..." >&2
    curl -sL "https://github.com/AppImage/AppImageKit/releases/download/continuous/appimagetool-x86_64.AppImage" -o "$tool"
    chmod 0700 "$tool"
  fi
  echo "$tool"
}

package_appimage() {
  local arch=$1
  local binary=$2
  local out=$3
  local tool
  tool=$(appimagetool_path) || return 0
  local appdir="/tmp/boxd-appdir-${arch}"
  local icon="$root_dir/desktop/build/appicon.png"

  rm -rf "$appdir"
  mkdir -p "$appdir/usr/bin" "$appdir/usr/share/applications" "$appdir/usr/share/icons/hicolor/256x256/apps"
  install -m 0700 "$binary" "$appdir/usr/bin/boxd"

  cat >"$appdir/AppRun" <<'APPEOF'
#!/usr/bin/env bash
# headless boxd AppImage 入口：参数透传给 boxd 二进制。
exec "$(dirname "$(readlink -f "$0")")/usr/bin/boxd" "$@"
APPEOF
  chmod 0700 "$appdir/AppRun"

  cat >"$appdir/boxd.desktop" <<'DESKEOF'
[Desktop Entry]
Type=Application
Name=boxd
Comment=single-node control plane for sing-box
Exec=boxd %F
Icon=boxd
Categories=Network;Utility;
Terminal=true
DESKEOF
  cp "$appdir/boxd.desktop" "$appdir/usr/share/applications/boxd.desktop"

  if [[ -f "$icon" ]]; then
    cp "$icon" "$appdir/usr/share/icons/hicolor/256x256/apps/boxd.png"
    cp "$icon" "$appdir/boxd.png"
  fi

  export APPIMAGE_EXTRACT_AND_RUN=1
  timeout 120 "$tool" "$appdir" "$out/boxd_${version}_linux_${arch}.AppImage" >/dev/null 2>&1 || \
    echo "AppImage packaging failed for $arch (appimagetool missing deps)" >&2
  unset APPIMAGE_EXTRACT_AND_RUN
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

echo "Headless Linux packages ready in $root_dir/build/linux/bin"
