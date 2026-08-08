#!/usr/bin/env bash
set -euo pipefail

# lib-version_test.sh 验证 nightly 版本解析（deb/rpm 版本合法且单调递增）。
root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
source "$root_dir/scripts/lib-version.sh"

nightly=$(resolve_package_version nightly)

# nightly 版本形如 0.1.0.<YYYYMMDD>.<HHMMSS>：以数字开头、仅含数字与点号。
if [[ ! "$nightly" =~ ^0\.1\.0\.[0-9]{8}\.[0-9]{6}$ ]]; then
  echo "unexpected nightly version: $nightly" >&2
  exit 1
fi

# 非 nightly 输入原样保留（去除 v 前缀，与 nfpm 约定一致）。
[[ "$(resolve_package_version v0.1.0)" == "0.1.0" ]]
[[ "$(resolve_package_version 0.2.3)" == "0.2.3" ]]

# 新 nightly 版本必须大于旧 nightly 版本（dpkg/rpm 语义）。
older=$(resolve_package_version nightly)
newer="${older%.*}.999999"
if [[ ! "$newer" > "$older" ]]; then
  echo "version ordering failed: $newer <= $older" >&2
  exit 1
fi

echo "lib-version.sh OK: nightly version resolved to $nightly"