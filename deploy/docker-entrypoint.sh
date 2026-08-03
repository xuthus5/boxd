#!/usr/bin/env sh
set -eu

# docker-entrypoint.sh —— boxd 容器入口。
# ICMP 测速需要原始 socket（CAP_NET_RAW）：容器默认以 root 运行，
# 配合 docker run --cap-add NET_RAW 生效（见 README「Docker」章节）。
# 本脚本仅完成数据目录初始化后启动 boxd。

# 数据目录初始化（容器卷可能以 root 挂载，boxd 内部按需使用）。
mkdir -p /var/lib/boxd /etc/sing-box

exec /app/boxd "$@"
