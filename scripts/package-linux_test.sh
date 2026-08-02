#!/usr/bin/env bash
set -euo pipefail

# package-linux_test.sh 验证 headless boxd 的 deb/rpm/AppImage 打包。
root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
output_dir=$(mktemp -d)
trap 'rm -rf "$output_dir"' EXIT

version=v0.1.0-test
bin_dir="$root_dir/build/linux/bin"
rm -rf "$bin_dir"

# 仅 amd64：AppImage 工具链仅 x86_64 宿主可用。
"$root_dir/scripts/package-linux.sh" "$version" amd64

# 三种包都存在且可执行。
test -x "$bin_dir/boxd_${version}_linux_amd64.AppImage"
test -f "$bin_dir/boxd_${version}_linux_amd64.deb"
test -f "$bin_dir/boxd_${version}_linux_amd64.rpm"

# AppImage 透传参数可运行。
appimage_version=$(APPIMAGE_EXTRACT_AND_RUN=1 "$bin_dir/boxd_${version}_linux_amd64.AppImage" --version 2>/dev/null) || true
echo "$appimage_version" | grep -q "$version"

# rpm 内容包含关键文件。
rpm_list=$(rpm -qlp "$bin_dir/boxd_${version}_linux_amd64.rpm" 2>/dev/null) || true
echo "$rpm_list" | grep -q "/usr/local/bin/boxd"
echo "$rpm_list" | grep -q "/lib/systemd/system/boxd.service"
echo "$rpm_list" | grep -q "/usr/lib/boxd/postinstall.sh"

# deb 内容包含关键文件（用 ar + tar 检查）。
tmpdir=$(mktemp -d)
trap 'rm -rf "$output_dir" "$tmpdir"' EXIT
(cd "$tmpdir" && ar x "$bin_dir/boxd_${version}_linux_amd64.deb")
deb_list=$(tar -tzf "$tmpdir/data.tar.gz" 2>/dev/null) || true
echo "$deb_list" | grep -q "./usr/local/bin/boxd"
echo "$deb_list" | grep -q "./lib/systemd/system/boxd.service"

echo "package-linux.sh OK: deb/rpm/AppImage verified"
