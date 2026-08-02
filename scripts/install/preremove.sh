#!/usr/bin/env bash
set -euo pipefail

# preremove.sh —— boxd deb/rpm 卸载前处理。
# 停止并禁用 systemd 服务，保留数据目录（/var/lib/boxd）供用户决策。

if command -v systemctl >/dev/null 2>&1; then
  systemctl stop boxd.service >/dev/null 2>&1 || true
  systemctl disable boxd.service >/dev/null 2>&1 || true
fi

exit 0
