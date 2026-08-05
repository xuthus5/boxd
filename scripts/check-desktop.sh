#!/usr/bin/env bash
set -euo pipefail

# check-desktop.sh 桌面应用（desktop/ 独立 Go module）质量门禁。
# 运行：测试、race、覆盖率、golangci-lint、govulncheck。

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
coverage_file=$(mktemp)
trap 'rm -f "$coverage_file"' EXIT

export PATH="$PATH:/usr/local/go/bin:$HOME/go/bin"
export GOPROXY="${GOPROXY:-https://proxy.golang.org,https://goproxy.io,direct}"

cd "$root_dir/desktop"

echo "==> go test"
go test ./...

echo "==> go test -race"
go test -race ./...

echo "==> go test -cover"
go test -coverprofile="$coverage_file" ./...

coverage=$(awk 'NR > 1 { total += $2; if ($3 > 0) covered += $2 } END { print 100 * covered / total }' "$coverage_file")
# desktop 是 GUI 壳：窗口/托盘/对话框等代码无法在无 GUI 环境单测，覆盖率门禁低于纯逻辑主模块。
awk -v coverage="$coverage" 'BEGIN { if (coverage + 0 < 60) exit 1 }'
printf 'Go statement coverage: %.2f%%\n' "$coverage"

echo "==> golangci-lint"
golangci-lint run ./...

echo "==> govulncheck"
govulncheck ./...
