#!/usr/bin/env bash
set -euo pipefail

INSTALL_BIN=/usr/local/bin/boxd
SYSTEMD_DIR=/etc/systemd/system
UNIT_NAME=boxd.service
UNIT_PATH="${SYSTEMD_DIR}/${UNIT_NAME}"
DATA_DIR=/var/lib/boxd

usage() {
  cat <<'EOF'
Usage: sudo ./uninstall.sh [options]

Remove the boxd binary and systemd service. Application data is retained by
default so that configuration and credentials are not deleted accidentally.

Options:
  --purge      Also delete /var/lib/boxd
  -h, --help   Show this help
EOF
}

purge=false
while (($# > 0)); do
  case "$1" in
    --purge)
      purge=true
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

if [[ "${EUID}" -ne 0 ]]; then
  echo "boxd uninstallation must run as root" >&2
  exit 1
fi

if command -v systemctl >/dev/null 2>&1; then
  systemctl disable --now "${UNIT_NAME}" >/dev/null 2>&1 || true
fi

rm -f -- "${UNIT_PATH}" "${INSTALL_BIN}"

if command -v systemctl >/dev/null 2>&1; then
  systemctl daemon-reload
  systemctl reset-failed "${UNIT_NAME}" >/dev/null 2>&1 || true
fi

if [[ "${purge}" == true ]]; then
  rm -rf -- "${DATA_DIR}"
  echo "boxd removed, including application data in ${DATA_DIR}"
else
  echo "boxd removed; application data was retained in ${DATA_DIR}"
  echo "use --purge to remove it as well"
fi
