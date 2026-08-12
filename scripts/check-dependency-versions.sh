#!/usr/bin/env bash
set -euo pipefail

# check-dependency-versions.sh 校验项目锁定的依赖版本是否为最新。
# 当前检查：sing-box（根 go.mod 直接依赖）、wails/v3（desktop/go.mod 直接依赖）。
# 任一锁定版本落后于官方最新 release 即返回非零。
#
# 用法: ./scripts/check-dependency-versions.sh
# 依赖: curl（访问 GitHub API / npm registry），网络需可达外网。

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
failed=0

# 有 GITHUB_TOKEN 时携带认证头，规避匿名 API 限流（CI 场景）。
CURL_AUTH=()
if [ -n "${GITHUB_TOKEN:-}" ]; then
  CURL_AUTH=(-H "Authorization: Bearer ${GITHUB_TOKEN}")
fi

# latest_tag <repo> <mode>
# mode=stable 使用 releases/latest（最新正式版）；mode=prerelease 使用列表中最新的预发布版。
latest_tag() {
  local repo="$1" mode="${2:-stable}"
  if [ "$mode" = "stable" ]; then
    curl --fail --silent --show-error --location "${CURL_AUTH[@]}" \
      "https://api.github.com/repos/${repo}/releases/latest" \
      | python3 -c "import json, sys; print(json.load(sys.stdin).get('tag_name', ''))"
    return
  fi
  curl --fail --silent --show-error --location "${CURL_AUTH[@]}" \
    "https://api.github.com/repos/${repo}/releases?per_page=15" \
    | python3 -c "
import json, sys
releases = json.load(sys.stdin)
for r in releases:
    if r.get('draft'):
        continue
    if not r.get('prerelease'):
        continue
    print(r['tag_name'])
    break
"
}

# pinned_version <module> <go.mod>
pinned_version() {
  local module="$1" modfile="$2"
  awk -v mod="$module" '$1 == mod { print $2; exit }' "$modfile"
}

check_github_dep() {
  local name="$1" module="$2" modfile="$3" repo="$4" mode="${5:-stable}" prefix="${6:-}"
  local pinned latest
  pinned=$(pinned_version "$module" "$modfile")
  latest=$(latest_tag "$repo" "$mode")
  if [ -z "$pinned" ] || [ -z "$latest" ]; then
    echo "ERROR: ${name}: pinned='${pinned}' latest='${latest}' (unresolvable)" >&2
    failed=1
    return
  fi
  latest=${latest#"$prefix"}
  pinned=${pinned#"$prefix"}
  if [ "$pinned" != "$latest" ]; then
    echo "FAIL: ${name} pinned ${pinned}, latest is ${latest}" >&2
    failed=1
  else
    echo "OK: ${name} ${pinned} is up to date"
  fi
}

check_github_dep "sing-box" \
  "github.com/sagernet/sing-box" "$root_dir/go.mod" \
  "SagerNet/sing-box" "stable" "v"

check_github_dep "wails/v3" \
  "github.com/wailsapp/wails/v3" "$root_dir/desktop/go.mod" \
  "wailsapp/wails" "prerelease" "v"

if [ "$failed" -ne 0 ]; then
  echo "Dependency versions are outdated; run 'go get' to upgrade." >&2
  exit 1
fi
echo "All tracked dependency versions are up to date."
