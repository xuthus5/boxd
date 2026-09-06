#!/usr/bin/env bash
set -euo pipefail

# build-installer.sh 构建双架构 NSIS 安装包。
# 在 Makefile build-desktop-windows 目标中调用。

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 查找 makensis
if command -v makensis >/dev/null 2>&1; then
  MAKENSIS="makensis"
elif [ -f "/c/Program Files (x86)/NSIS/makensis.exe" ]; then
  MAKENSIS="/c/Program Files (x86)/NSIS/makensis.exe"
elif [ -f "/c/Program Files/NSIS/makensis.exe" ]; then
  MAKENSIS="/c/Program Files/NSIS/makensis.exe"
else
  echo "Error: makensis not found" >&2
  exit 1
fi

cd "$SCRIPT_DIR"
"$MAKENSIS" \
  -DARG_WAILS_AMD64_BINARY="..\..\..\bin\boxd-desktop-amd64.exe" \
  -DARG_WAILS_ARM64_BINARY="..\..\..\bin\boxd-desktop-arm64.exe" \
  -DINFO_PRODUCTVERSION="0.0.0.0" \
  project.nsi
