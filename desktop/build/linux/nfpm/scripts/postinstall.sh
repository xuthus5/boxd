#!/bin/sh
# boxd-desktop 安装后脚本：为二进制授予 ICMP 测速所需的原始套接字能力。
# 以普通用户运行时无需 root 即可执行 ICMP 测速（等价 systemd 的 AmbientCapabilities）。
set -eu

binary="/usr/local/bin/boxd-desktop"
if command -v setcap >/dev/null 2>&1; then
  if ! setcap cap_net_raw+ep "$binary" 2>/dev/null; then
    echo "warning: failed to grant CAP_NET_RAW to $binary; ICMP 测速将不可用" >&2
  fi
else
  echo "warning: setcap not found; 请运行 scripts/grant-desktop-icmp.sh 授权 ICMP 测速" >&2
fi
