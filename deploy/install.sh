#!/usr/bin/env bash
set -euo pipefail

INSTALL_BIN=/usr/local/bin/boxd
SYSTEMD_DIR=/etc/systemd/system
UNIT_NAME=boxd.service
UNIT_PATH="${SYSTEMD_DIR}/${UNIT_NAME}"
DATA_DIR=/var/lib/boxd

usage() {
  cat <<'EOF'
Usage: sudo ./install.sh [options]

Install or upgrade boxd and its systemd service.

Options:
  --no-start   Install and enable the service without starting it
  -h, --help   Show this help

Environment:
  BOXD_BINARY      Path to the boxd binary (default: ./boxd or ../bin/boxd)
  BOXD_UNIT_FILE   Path to boxd.service (default: next to this script)
  BOXD_START_NOW   true or false; overridden by --no-start
EOF
}

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
start_now="${BOXD_START_NOW:-true}"

while (($# > 0)); do
  case "$1" in
    --no-start)
      start_now=false
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
done

case "${start_now,,}" in
  true|false)
    start_now="${start_now,,}"
    ;;
  *)
    echo "BOXD_START_NOW must be true or false" >&2
    exit 2
    ;;
esac

if [[ "${EUID}" -ne 0 ]]; then
  echo "boxd installation must run as root" >&2
  exit 1
fi
if ! command -v systemctl >/dev/null 2>&1; then
  echo "systemctl is required; boxd release packages target systemd Linux hosts" >&2
  exit 1
fi

source_binary="${BOXD_BINARY:-}"
if [[ -z "${source_binary}" ]]; then
  if [[ -x "${script_dir}/boxd" ]]; then
    source_binary="${script_dir}/boxd"
  elif [[ -x "${script_dir}/../bin/boxd" ]]; then
    source_binary="${script_dir}/../bin/boxd"
  else
    echo "boxd binary not found; set BOXD_BINARY or run from a release package" >&2
    exit 1
  fi
fi
if [[ ! -x "${source_binary}" ]]; then
  echo "boxd binary is not executable: ${source_binary}" >&2
  exit 1
fi

source_unit="${BOXD_UNIT_FILE:-${script_dir}/boxd.service}"
if [[ ! -f "${source_unit}" ]]; then
  echo "systemd unit not found: ${source_unit}" >&2
  exit 1
fi

if ! version_output="$("${source_binary}" -version 2>&1)"; then
  echo "boxd binary cannot run on this host: ${version_output}" >&2
  exit 1
fi

install -d -m 0755 "$(dirname "${INSTALL_BIN}")"
install -d -m 0750 "${DATA_DIR}"

binary_temp="$(mktemp "${INSTALL_BIN}.XXXXXX")"
unit_temp="$(mktemp "${SYSTEMD_DIR}/.${UNIT_NAME}.XXXXXX")"
cleanup() {
  rm -f -- "${binary_temp}" "${unit_temp}"
}
trap cleanup EXIT

install -m 0755 "${source_binary}" "${binary_temp}"
install -m 0644 "${source_unit}" "${unit_temp}"
"${binary_temp}" -version >/dev/null

mv -f -- "${binary_temp}" "${INSTALL_BIN}"
mv -f -- "${unit_temp}" "${UNIT_PATH}"
binary_temp=""
unit_temp=""

systemctl daemon-reload
systemctl enable "${UNIT_NAME}" >/dev/null

if [[ "${start_now}" == true ]]; then
  systemctl restart "${UNIT_NAME}"
  echo "boxd installed and started: ${version_output}"
  echo "web console: http://127.0.0.1:7662/"
  echo "service logs: journalctl -u ${UNIT_NAME} -f"
else
  echo "boxd installed and enabled but not started: ${version_output}"
  echo "start it with: systemctl start ${UNIT_NAME}"
fi
