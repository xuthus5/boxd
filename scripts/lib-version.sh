#!/usr/bin/env bash
set -euo pipefail

# resolve_package_version 解析打包版本号。
# nightly 构建映射为 0.1.0.<UTC日期>.<UTC时间>：纯数字与点号，deb/rpm 均合法；
# 每次构建版本单调递增，dnf/yum upgrade 可识别新版本，避免同版本重复安装报错；
# 基线取当前桌面端 nfpm 的 0.1.0，保证用户可从旧 nightly 包（"nightly"/"0.1.0"）平滑升级。
# 其余版本保持原样并去除 v 前缀（与 nfpm 版本约定一致）。
resolve_package_version() {
  local version=$1
  if [[ "$version" == "nightly" ]]; then
    printf '0.1.0.%s.%s' "$(date -u +%Y%m%d)" "$(date -u +%H%M%S)"
  else
    printf '%s' "${version#v}"
  fi
}