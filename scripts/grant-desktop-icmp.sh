#!/usr/bin/env bash
set -euo pipefail

# grant-desktop-icmp.sh 为桌面应用授权 ICMP 测速所需的网络能力，
# 免去以 root 运行 boxd-desktop / 不依赖 systemd AmbientCapabilities。
#
# 用法（需要 root）:
#   sudo ./scripts/grant-desktop-icmp.sh [用户名] [sysctl|setcap]
#
#   USER     要授权的 Unix 用户名（默认取当前 sudo 调用者 / 登录用户）。
#            脚本实时解析其主 GID，不硬编码。
#   方法     sysctl（默认）：写入 /etc/sysctl.d/99-boxd-icmp.conf
#            net.ipv4.ping_group_range = <gid> <gid> 并立即生效，
#            允许该 GID 内进程以普通权限打开 ICMP 原始套接字。
#            setcap：对该用户的二进制 /usr/local/bin/boxd-desktop 直接
#            授予 cap_net_raw（仅对该文件生效，需重新打包或安装后重新执行）。

die() {
  echo "error: $*" >&2
  exit 1
}

[[ $EUID -eq 0 ]] || die "需要 root 权限：sudo $0 $*"

user_name="${1:-${SUDO_USER:-$USER}}"
method="${2:-sysctl}"
binary="/usr/local/bin/boxd-desktop"

if [[ $user_name == "root" ]]; then
  die "请指定实际的桌面用户（root 无需授权）"
fi
if ! getent passwd "$user_name" >/dev/null 2>&1; then
  die "用户 $user_name 不存在"
fi
gid=$(id -g "$user_name")

case "$method" in
sysctl)
  conf="/etc/sysctl.d/99-boxd-icmp.conf"
  printf 'net.ipv4.ping_group_range = %s %s\n' "$gid" "$gid" >"$conf"
  sysctl -w "net.ipv4.ping_group_range=$gid $gid" >/dev/null
  echo "已授权 ICMP: 用户 $user_name (GID $gid) 的普通进程可发起 ping，配置已写入 $conf"
  ;;
setcap)
  [[ -x "$binary" ]] || die "未找到 $binary，请先安装 boxd-desktop 包"
  if ! command -v setcap >/dev/null 2>&1; then
    die "未找到 setcap 命令，请安装 libcap-bin/procps 或改用默认的 sysctl 方法"
  fi
  setcap cap_net_raw+ep "$binary"
  echo "已授权 ICMP: $binary 持有 cap_net_raw"
  ;;
*)
  die "未知方法 $method（可选: sysctl / setcap）"
  ;;
esac

echo "验证:"
if command -v getcap >/dev/null 2>&1; then
  getcap "$binary" 2>/dev/null || true
fi
echo "  sysctl net.ipv4.ping_group_range = $(sysctl -n net.ipv4.ping_group_range 2>/dev/null || echo 不可读)"