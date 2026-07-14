#!/usr/bin/env bash
set -euo pipefail

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
coverage_file=$(mktemp)
trap 'rm -f "$coverage_file"' EXIT

cd "$root_dir"
mapfile -t packages < <(go list ./... | grep -v '/ui/node_modules/')

go test "${packages[@]}"
go test -race "${packages[@]}"
go test -coverprofile="$coverage_file" "${packages[@]}"

coverage=$(awk 'NR > 1 { total += $2; if ($3 > 0) covered += $2 } END { print 100 * covered / total }' "$coverage_file")
awk -v coverage="$coverage" 'BEGIN { if (coverage + 0 < 90) exit 1 }'
printf 'Go statement coverage: %.2f%%\n' "$coverage"

golangci-lint run ./...
govulncheck "${packages[@]}"
