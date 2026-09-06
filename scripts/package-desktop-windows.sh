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
kernel_version="${KERNEL_VERSION:-1.13.21}"
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

echo "==> Generating Windows resource (.syso) for icon"
syso_file="boxd-desktop_windows_${arch}.syso"
wails3 generate syso -arch "$arch" -icon "build/boxd-desktop.ico" -manifest "build/boxd-desktop.manifest" -out "$syso_file" 2>/dev/null || {
  echo "==> wails3 generate syso failed (non-fatal, continuing without icon)"
  syso_file=""
}

echo "==> Building Windows desktop binary (${arch})"
mkdir -p bin
if [[ -n "$syso_file" ]]; then
  trap 'rm -f "$syso_file"' EXIT
fi
# 正式发布构建（非 nightly）启用 Wails production tag，禁用 WebView 开发者工具；
# nightly 保留开发工具便于排查。
build_tags="desktop embed_ui with_gvisor with_quic with_dhcp with_wireguard with_utls with_acme with_clash_api"
if [[ "$version" != "nightly" ]]; then
  build_tags="$build_tags production"
fi
go build \
  -tags "$build_tags" \
  -ldflags "-w -s -X github.com/xuthus5/boxd/internal/core.Version=$version -X github.com/sagernet/sing-box/constant.Version=$kernel_version" \
  -o bin/boxd-desktop.exe ./

echo "==> Generating NSIS installer"
nsis_dir="$root_dir/desktop/build/windows/nsis"
# 将 MSYS2 路径转换为 Windows 原生格式，NSIS 的 File 命令需要反斜杠路径
binary_path_win=$(cygpath -w "$root_dir/desktop/bin/boxd-desktop.exe")
# 查找 makensis
makensis_bin=""
if command -v makensis >/dev/null 2>&1; then
  makensis_bin="makensis"
elif [[ -f "/c/Program Files (x86)/NSIS/makensis.exe" ]]; then
  makensis_bin="/c/Program Files (x86)/NSIS/makensis.exe"
elif [[ -f "/c/Program Files/NSIS/makensis.exe" ]]; then
  makensis_bin="/c/Program Files/NSIS/makensis.exe"
fi
if [[ -n "$makensis_bin" ]]; then
  # 生成 WebView2 引导程序
  wails3 generate webview2bootstrapper -dir "$nsis_dir" >/dev/null 2>&1 || echo "==> WebView2 bootstrapper generation skipped"
  # 根据架构设置 NSIS 变量
  arch_upper=$(echo "$arch" | tr '[:lower:]' '[:upper:]')
  # NSIS 要求版本号为 X.X.X.X 数字格式，nightly 等非数字版本转为 0.0.0.0
  nsis_version=$(echo "$version" | sed -E 's/^[^0-9]*/0.0.0./; s/[^0-9.]/-/g')
  if ! echo "$nsis_version" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$'; then
    nsis_version="0.0.0.0"
  fi
  # 调用 NSIS 生成安装程序
  installer_name="boxd-desktop-${arch}-installer.exe"
  "$makensis_bin" -DARG_WAILS_${arch_upper}_BINARY="$binary_path_win" \
    -DINFO_PROJECTNAME="boxd-desktop" \
    -DINFO_COMPANYNAME="boxd developers" \
    -DINFO_PRODUCTNAME="boxd desktop" \
    -DINFO_PRODUCTVERSION="$nsis_version" \
    -DINFO_COPYRIGHT="$(date +%Y) boxd developers" \
    "$nsis_dir/project.nsi" && {
    # 压缩安装程序
    zip_name="boxd-desktop-${arch}-installer.zip"
    if command -v zip >/dev/null 2>&1; then
      (cd "$root_dir/desktop/bin" && zip -q -9 "$root_dir/$zip_name" "$installer_name")
    elif command -v 7z >/dev/null 2>&1; then
      (cd "$root_dir/desktop/bin" && 7z a -tzip "$root_dir/$zip_name" "$installer_name" >/dev/null)
    elif [ -f "/c/Program Files/7-Zip/7z.exe" ]; then
      (cd "$root_dir/desktop/bin" && "/c/Program Files/7-Zip/7z.exe" a -tzip "$root_dir/$zip_name" "$installer_name" >/dev/null)
    fi
    if [[ -f "$root_dir/$zip_name" ]]; then
      echo "==> Output: $root_dir/$zip_name"
      ls -lh "$root_dir/$zip_name"
    fi
  } || echo "==> NSIS build failed (non-fatal)"
else
  echo "==> makensis not found, skipping NSIS installer"
fi

# 代码签名（可选）
sign_installer() {
  local file="$1"
  if [ -z "${CODESIGN_CERT:-}" ] || [ -z "${CODESIGN_PASSWORD:-}" ]; then
    echo "==> Code signing skipped (CODESIGN_CERT or CODESIGN_PASSWORD not set)"
    return 0
  fi
  if ! command -v signtool >/dev/null 2>&1 && ! command -v osslsigncode >/dev/null 2>&1; then
    echo "==> Code signing skipped (signtool or osslsigncode not found)"
    return 0
  fi
  echo "==> Signing $file"
  if command -v signtool >/dev/null 2>&1; then
    signtool sign /f "$CODESIGN_CERT" /p "$CODESIGN_PASSWORD" /tr "${TIMESTAMP_URL:-http://timestamp.digicert.com}" /td sha256 /fd sha256 "$file"
  elif command -v osslsigncode >/dev/null 2>&1; then
    osslsigncode sign -pkcs12 "$CODESIGN_CERT" -pass "$CODESIGN_PASSWORD" -tr "${TIMESTAMP_URL:-http://timestamp.digicert.com}" -in "$file" -out "${file}-signed"
    mv "${file}-signed" "$file"
  fi
}
# 签名主二进制
sign_installer "$root_dir/desktop/bin/boxd-desktop.exe"
# 签名安装程序
installer_file=$(find "$root_dir/desktop/bin" -name "*-installer.exe" -type f | head -1)
if [ -n "$installer_file" ]; then
  sign_installer "$installer_file"
fi

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
