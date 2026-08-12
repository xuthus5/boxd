#!/usr/bin/env bash
set -euo pipefail

# update-nightly-release.sh 重建 nightly GitHub Release：
# 1. 生成平台/产物分区的可视化发布说明（含直接下载链接与校验说明）
# 2. 删除旧 release（连带旧产物与 tag），以最新提交重建
#
# 用法: ./scripts/update-nightly-release.sh <repo> <commit-sha> [target-sha]
#   repo        如 xuthus5/boxd
#   commit-sha 本次构建对应的提交
#   target-sha  指向 release 的提交（默认同 commit-sha）
# 依赖: gh, git；环境变量 GITHUB_TOKEN 需具备 contents:write 权限。
# 产物命名须与各打包脚本一致（nightly 固定命名），缺失的产物链接对应 404。

repo="${1:?repo required (e.g. xuthus5/boxd)}"
sha="${2:?commit sha required}"
target="${3:-$sha}"
if [ "${NOTES_ONLY:-0}" != "1" ] && ! command -v gh >/dev/null 2>&1; then
  echo "gh CLI is required (or set NOTES_ONLY=1 to preview notes)" >&2
  exit 1
fi

notes_file=$(mktemp)
trap 'rm -f "$notes_file"' EXIT

repo_url="https://github.com/${repo}"
dl_url="${repo_url}/releases/download/nightly"
commit_url="${repo_url}/commit/${sha}"
build_time=$(date -u '+%Y-%m-%d %H:%M UTC')

# 资产链接辅助：dl <资产名>
dl() { printf '[⬇ 下载](%s/%s)' "$dl_url" "$1"; }
# 校验链接：dlsha <资产名>
dlsha() { printf '[sha256](%s/%s.sha256)' "$dl_url" "$1"; }

cat >"$notes_file" <<EOF
## 🚀 boxd Nightly Build

| 构建时间 | 提交 | 分支 |
| --- | --- | --- |
| \`${build_time}\` | [\`${sha:0:12}\`](${commit_url}) | \`main\` |

> ⚠️ **预发布版本**：由 \`main\` 分支自动构建，包含最新变更，可能存在不稳定改动，仅供测试。

---

## 📦 产物

### 🐧 Linux

**Headless 服务**（无界面 Web 服务，适合服务器/容器）

| 架构 | 下载 | 校验 |
| --- | --- | --- |
| amd64 | $(dl boxd_nightly_linux_amd64.tar.gz) | $(dlsha boxd_nightly_linux_amd64.tar.gz) |
| arm64 | $(dl boxd_nightly_linux_arm64.tar.gz) | $(dlsha boxd_nightly_linux_arm64.tar.gz) |

**Headless 服务 deb / rpm / AppImage**

| 架构 | deb | rpm | AppImage |
| --- | --- | --- | --- |
| amd64 | $(dl boxd_nightly_linux_amd64.deb) | $(dl boxd_nightly_linux_amd64.rpm) | $(dl boxd_nightly_linux_amd64.AppImage) |
| arm64 | $(dl boxd_nightly_linux_arm64.deb) | $(dl boxd_nightly_linux_arm64.rpm) | $(dl boxd_nightly_linux_arm64.AppImage) |

**桌面应用**（boxd-desktop, Wails3 + GTK4）

| 架构 | 二进制 | deb | rpm | AppImage |
| --- | --- | --- | --- | --- |
| amd64 | $(dl boxd-desktop-linux-amd64) | $(dl boxd-desktop-linux-amd64.deb) | $(dl boxd-desktop-linux-amd64.rpm) | $(dl boxd-desktop-linux-amd64.AppImage) |
| arm64 | $(dl boxd-desktop-linux-arm64) | $(dl boxd-desktop-linux-arm64.deb) | $(dl boxd-desktop-linux-arm64.rpm) | $(dl boxd-desktop-linux-arm64.AppImage) |

### 🪟 Windows

**Headless 服务**

| 架构 | 下载 | 校验 |
| --- | --- | --- |
| amd64 | $(dl boxd_nightly_windows_amd64.tar.gz) | $(dlsha boxd_nightly_windows_amd64.tar.gz) |
| arm64 | $(dl boxd_nightly_windows_arm64.tar.gz) | $(dlsha boxd_nightly_windows_arm64.tar.gz) |

**桌面应用**

| 架构 | 下载 |
| --- | --- |
| amd64 | $(dl boxd-desktop-nightly-windows-amd64.zip) |

### 🍎 macOS

**Headless 服务**

| 架构 | 下载 | 校验 |
| --- | --- | --- |
| amd64 | $(dl boxd_nightly_darwin_amd64.tar.gz) | $(dlsha boxd_nightly_darwin_amd64.tar.gz) |
| arm64 | $(dl boxd_nightly_darwin_arm64.tar.gz) | $(dlsha boxd_nightly_darwin_arm64.tar.gz) |

**桌面应用**

| 架构 | zip | dmg |
| --- | --- | --- |
| arm64 | $(dl boxd-desktop-nightly-macos-arm64.zip) | $(dl boxd-desktop-nightly-macos-arm64.dmg) |
| amd64 | $(dl boxd-desktop-nightly-macos-amd64.zip) | $(dl boxd-desktop-nightly-macos-amd64.dmg) |

### 🐳 Docker

\`\`\`bash
docker pull ghcr.io/${repo,,}:nightly
\`\`\`

---

## ✅ 校验

所有 tar.gz 产物附带 \`.sha256\` 校验文件，下载后可用以下命令验证：

\`\`\`bash
sha256sum -c boxd_nightly_linux_amd64.tar.gz.sha256
\`\`\`

## 📜 版本信息

- 提交：[\`${sha:0:12}\`](${commit_url})
- 构建时间：\`${build_time}\`（UTC）
- 此 release 始终指向 \`main\` 分支的最新成功构建，每次构建会重建并替换。
EOF

echo "==> Generated nightly release notes (${repo} @ ${sha:0:12})"

if [ "${NOTES_ONLY:-0}" = "1" ]; then
  cat "$notes_file"
  exit 0
fi

# 重建 release：先删除旧 release 与关联 tag，再以最新提交创建 draft release。
# draft 状态保证所有 job 完成资产上传前用户看不到半成品；由 finalize job 统一发布。
if gh release view nightly --repo "$repo" >/dev/null 2>&1; then
  gh release delete nightly --repo "$repo" --yes --cleanup-tag
fi
git tag -f nightly "$target"
git push -f origin refs/tags/nightly
gh release create nightly \
  --repo "$repo" \
  --title "Nightly" \
  --prerelease \
  --draft \
  --notes-file "$notes_file" \
  --target "$target"
echo "==> Nightly release recreated (draft; published by finalize job after asset upload)"
