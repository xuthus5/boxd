#!/usr/bin/env bash
set -euo pipefail

# postinstall.sh —— boxd deb/rpm 安装后配置。
# 创建服务用户与数据/配置目录，启用并启动 systemd 服务。

# 创建 boxd 系统用户（若不存在）。
if ! id boxd >/dev/null 2>&1; then
  useradd --system --home /var/lib/boxd --shell /sbin/nologin boxd || true
fi

install -d -o boxd -g boxd -m 0700 /var/lib/boxd
install -d -o root -g boxd -m 0750 /etc/boxd
install -d -o boxd -g boxd -m 0750 /etc/sing-box

# 提供默认 env（若用户未配置）。
if [[ ! -f /etc/boxd/boxd.env && -f /etc/boxd/boxd.env.example ]]; then
  install -o root -g boxd -m 0640 /etc/boxd/boxd.env.example /etc/boxd/boxd.env
fi

if command -v systemctl >/dev/null 2>&1; then
  systemctl daemon-reload >/dev/null 2>&1 || true
  systemctl enable boxd.service >/dev/null 2>&1 || true
  systemctl restart boxd.service >/dev/null 2>&1 || true
fi

exit 0
