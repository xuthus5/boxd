#!/usr/bin/env bash
set -euo pipefail

# build-desktop.sh 构建 boxd-desktop 桌面应用。
# 用法: ./scripts/build-desktop.sh [VERSION] [GOOS]
# GOOS 默认为当前系统；可显式传入 windows 或 linux。

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
version="${1:-$(git -C "$root_dir" describe --tags --always --dirty 2>/dev/null || echo dev)}"
kernel_version="${KERNEL_VERSION:-1.13.21}"
target_os="${2:-$(go env GOOS)}"
export PATH="$PATH:/usr/local/go/bin:$HOME/go/bin"
export GOPROXY="${GOPROXY:-https://proxy.golang.org,https://goproxy.io,direct}"
export GOOS="$target_os"

# Linux 桌面依赖 GTK4/WebKitGTK（cgo），必须开启 CGO。
if [ "$target_os" = "linux" ]; then
  export CGO_ENABLED=1
else
  export CGO_ENABLED=0
fi

echo "==> Building frontend"
(cd "$root_dir/ui" && npm run build)

echo "==> Copying frontend dist to desktop/ui/dist"
mkdir -p "$root_dir/desktop/ui"
rm -rf "$root_dir/desktop/ui/dist"
cp -r "$root_dir/ui/dist" "$root_dir/desktop/ui/dist"
if [ "$target_os" != "windows" ]; then
  find "$root_dir/desktop/ui" -type d -exec chmod 0700 {} +
  find "$root_dir/desktop/ui" -type f -exec chmod 0600 {} +
fi

echo "==> Generating Wails bindings"
(cd "$root_dir/desktop" && wails3 generate bindings -d "$root_dir/ui/src/lib/api/bindings" >/dev/null)

echo "==> Building desktop binary ($target_os)"
mkdir -p "$root_dir/desktop/bin"

ldflags="-X github.com/xuthus5/boxd/internal/core.Version=$version -X github.com/sagernet/sing-box/constant.Version=$kernel_version"
if [ "$target_os" = "windows" ]; then
  ldflags="$ldflags -H=windowsgui"
  output="bin/boxd-desktop.exe"

  # 编译 Windows 资源（图标 + 版本信息），Go 会自动链接 .syso
  echo "==> Compiling Windows resources"
  (cd "$root_dir/desktop" && windres -o app_windows.syso app.rc)
else
  output="bin/boxd-desktop"
fi

(cd "$root_dir/desktop" && go build \
  -tags "desktop embed_ui with_gvisor with_quic with_dhcp with_wireguard with_utls with_acme with_clash_api" \
  -ldflags "$ldflags" \
  -o "$output" ./)

echo "==> Built desktop/$output"

# Windows: 生成 NSIS 安装包
if [ "$target_os" = "windows" ]; then
  # 确保 makensis 可用
  nsis_dir="/c/Program Files (x86)/NSIS"
  if [ -d "$nsis_dir" ]; then
    export PATH="$nsis_dir:$PATH"
  fi
  if command -v makensis >/dev/null 2>&1; then
    # 下载 WebView2 引导程序（如不存在）
    nsis_dir_src="$root_dir/desktop/build/windows/nsis"
    webview2_exe="$nsis_dir_src/MicrosoftEdgeWebview2Setup.exe"
    if [ ! -f "$webview2_exe" ]; then
      echo "==> Downloading WebView2 bootstrapper"
      curl -sL -o "$webview2_exe" "https://go.microsoft.com/fwlink/p/?LinkId=2124703"
    fi
    echo "==> Building NSIS installer"
    (cd "$nsis_dir_src" && makensis \
      -DARG_WAILS_AMD64_BINARY="..\..\..\bin\boxd-desktop.exe" \
      project.nsi)
    echo "==> Built desktop/bin/boxd-desktop-amd64-installer.exe"
  else
    echo "==> Skipping NSIS installer (makensis not found)"
  fi
fi
