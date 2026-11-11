#!/usr/bin/env bash
set -euo pipefail
umask 077

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/boxd-mygo-view.XXXXXX")
app_pid=
xvfb_pid=
vnc_pid=
display=
port=
password_file="$work/vnc.pass"

usage() {
  cat <<'EOF'
Usage: ./scripts/mygo-remote-view.sh [--check]

Starts the MyGo desktop prototype in an isolated Xvfb session and, when
--check is not used, starts x11vnc bound to 127.0.0.1. The generated VNC
password and SSH tunnel command are printed once. Press Ctrl-C to stop.
EOF
}

# shellcheck disable=SC2329
cleanup() {
  if [ -n "$vnc_pid" ] && kill -0 "$vnc_pid" 2>/dev/null; then
    kill "$vnc_pid" 2>/dev/null || true
    wait "$vnc_pid" 2>/dev/null || true
  fi
  if [ -n "$app_pid" ] && kill -0 "$app_pid" 2>/dev/null; then
    kill "$app_pid" 2>/dev/null || true
    wait "$app_pid" 2>/dev/null || true
  fi
  if [ -n "$xvfb_pid" ] && kill -0 "$xvfb_pid" 2>/dev/null; then
    kill "$xvfb_pid" 2>/dev/null || true
    wait "$xvfb_pid" 2>/dev/null || true
  fi
  rm -rf -- "$work"
}
trap cleanup EXIT INT TERM

check_deps() {
  local missing=0
  for command in Xvfb x11vnc go python3; do
    if ! command -v "$command" >/dev/null 2>&1; then
      printf 'missing required command: %s\n' "$command" >&2
      missing=1
    fi
  done
  if [ "$missing" -ne 0 ]; then
    printf 'install x11vnc and Xvfb before using remote visual acceptance\n' >&2
    return 2
  fi
}

find_port() {
  python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1]); s.close()'
}

if [ "${1:-}" = "--check" ]; then
  check_deps
  printf 'remote visual dependencies are available\n'
  exit 0
fi
if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi
if [ "$#" -ne 0 ]; then
  usage >&2
  exit 2
fi

check_deps

mkdir -p "$work/home" "$work/data" "$work/config" "$work/runtime" "$work/logs"
chmod 700 "$work"

n=99
while [ -S "/tmp/.X11-unix/X$n" ]; do
  n=$((n + 1))
done
display=":$n"
Xvfb "$display" -screen 0 1440x900x24 -nolisten tcp > "$work/logs/xvfb.log" 2>&1 &
xvfb_pid=$!
for _ in $(seq 1 100); do
  [ -S "/tmp/.X11-unix/X$n" ] && break
  kill -0 "$xvfb_pid" 2>/dev/null || { printf 'Xvfb failed to start\n' >&2; exit 1; }
  sleep 0.1
done
[ -S "/tmp/.X11-unix/X$n" ] || { printf 'Xvfb display did not appear\n' >&2; exit 1; }

(
  cd "$root"
  GOTOOLCHAIN=auto go build -tags e2e -o "$work/boxd-view" .
)

HOME="$work/home" XDG_RUNTIME_DIR="$work/runtime" XDG_CACHE_HOME="$work/home/.cache" \
  GDK_BACKEND=x11 DISPLAY="$display" NO_AT_BRIDGE=1 \
  BOXD_DATA_DIR="$work/data" BOXD_CONFIG="$work/config/config.json" \
  BOXD_START_HIDDEN=0 BOXD_SKIP_KERNEL_AUTOSTART=1 \
  "$work/boxd-view" > "$work/logs/app.stdout" 2>&1 &
app_pid=$!

for _ in $(seq 1 200); do
  if ! kill -0 "$app_pid" 2>/dev/null; then
    printf 'boxd exited before the window was ready\n' >&2
    exit 1
  fi
  if command -v xdotool >/dev/null 2>&1; then
    DISPLAY="$display" xdotool search --name 'boxd' >/dev/null 2>&1 && break
  fi
  sleep 0.1
done

port=$(find_port)
password=$(python3 -c 'import secrets; print(secrets.token_urlsafe(18))')
x11vnc -storepasswd "$password" "$password_file" >/dev/null 2>&1
chmod 600 "$password_file"
x11vnc -display "$display" -localhost -rfbauth "$password_file" -rfbport "$port" \
  -forever -shared -noxdamage -nolookup > "$work/logs/x11vnc.log" 2>&1 &
vnc_pid=$!
vnc_ready=0
for _ in $(seq 1 100); do
  if ! kill -0 "$vnc_pid" 2>/dev/null; then
    printf 'x11vnc failed to start; see the temporary log\n' >&2
    exit 1
  fi
  if python3 - "$port" <<'PY'
import socket, sys
try:
    socket.create_connection(("127.0.0.1", int(sys.argv[1])), timeout=0.2).close()
except OSError:
    sys.exit(1)
PY
  then
    vnc_ready=1
    break
  fi
  sleep 0.1
done
if [ "$vnc_ready" -ne 1 ]; then
  printf 'x11vnc did not listen on 127.0.0.1:%s\n' "$port" >&2
  exit 1
fi

local_port=$(find_port)
cat <<EOF
MyGo remote visual session is ready.
VNC password: $password
Server loopback port: 127.0.0.1:$port
Open an SSH tunnel from your workstation:
  ssh -N -L $local_port:127.0.0.1:$port <user>@<server>
Then connect your VNC client to 127.0.0.1:$local_port.
Press Ctrl-C here to stop the application, Xvfb and x11vnc.
EOF

while kill -0 "$app_pid" 2>/dev/null && kill -0 "$vnc_pid" 2>/dev/null; do
  sleep 1
done

printf 'application or VNC server exited; cleaning up\n' >&2
exit 1
