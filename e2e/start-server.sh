#!/usr/bin/env bash
set -euo pipefail

root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
port="${BOXD_E2E_PORT:-17662}"
data_dir="${root}/test-results/e2e-data"
binary="${root}/test-results/bin/boxd-e2e"

if [[ ! "${port}" =~ ^[0-9]+$ ]]; then
  echo "BOXD_E2E_PORT must be numeric" >&2
  exit 2
fi

rm -rf -- "${data_dir}" "$(dirname -- "${binary}")"
mkdir -p -- "${data_dir}" "$(dirname -- "${binary}")"

cd "${root}"
CGO_ENABLED=0 go build -trimpath -tags "with_utls,with_quic" -o "${binary}" ./cmd/boxd
exec "${binary}" \
  -data-dir "${data_dir}" \
  -web \
  -listen "127.0.0.1:${port}"
