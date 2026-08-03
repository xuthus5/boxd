#!/usr/bin/env bash
set -euo pipefail

# docker-entrypoint_test.sh 验证容器 entrypoint：目录初始化与参数透传。
root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
sandbox=$(mktemp -d)
trap 'rm -rf "$sandbox"' EXIT

# 准备隔离的 boxd 与数据目录。
mkdir -p "$sandbox/app" "$sandbox/var/lib" "$sandbox/etc"
cat >"$sandbox/app/boxd" <<'FAKEEOF'
#!/usr/bin/env sh
echo "fake-boxd args: $*"
FAKEEOF
chmod 0700 "$sandbox/app/boxd"

# 动态替换脚本内的目录路径，指向隔离沙箱。
script=$(sed \
  -e "s|/var/lib/boxd|$sandbox/var/lib/boxd|g" \
  -e "s|/etc/sing-box|$sandbox/etc/sing-box|g" \
  -e "s|/app/boxd|$sandbox/app/boxd|g" \
  "$root_dir/deploy/docker-entrypoint.sh")

# 1. 非 root 路径：透传参数并调用 boxd（bash -c 首参为 $0，其余为 $@）。
bash -c "$script" entrypoint --version >"$sandbox/out1" 2>&1
grep -q "fake-boxd args: --version" "$sandbox/out1"

# 2. 目录初始化：非 root 也应创建数据目录。
test -d "$sandbox/var/lib/boxd"
test -d "$sandbox/etc/sing-box"

echo "docker-entrypoint.sh OK: init + arg passthrough verified"
