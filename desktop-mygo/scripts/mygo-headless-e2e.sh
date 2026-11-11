#!/usr/bin/env bash
set -euo pipefail
umask 077

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"
run_id=${MYGO_ACCEPTANCE_RUN_ID:-$(date +%Y%m%d-%H%M%S)}
evidence=${MYGO_ACCEPTANCE_DIR:-$root/artifacts/acceptance}/$run_id
work=$(mktemp -d "${TMPDIR:-/tmp}/boxd-mygo-e2e.XXXXXX")
app_pid=
xvfb_pid=
display=
app_socket=
automation_bin=${MYGO_AUTOMATION_BIN:-$(command -v mygo-automation || true)}
session=$(printf '%s' "boxd-e2e-$run_id" | tr -c 'A-Za-z0-9_.-' '-')

automation() {
  "$automation_bin" "$@" --session "$session"
}

mkdir -m 700 -p "$work/home" "$work/data" "$work/config" "$work/runtime" "$work/logs" "$evidence/logs" "$evidence/screenshots"
chmod 700 "$work" "$evidence"

cleanup() {
  if [ -n "${automation_bin:-}" ]; then
    automation close >/dev/null 2>&1 || true
  fi
  if [ -n "$app_pid" ] && kill -0 "$app_pid" 2>/dev/null; then
    kill "$app_pid" 2>/dev/null || true
    wait "$app_pid" 2>/dev/null || true
  fi
  if [ -n "$xvfb_pid" ] && kill -0 "$xvfb_pid" 2>/dev/null; then
    kill "$xvfb_pid" 2>/dev/null || true
    wait "$xvfb_pid" 2>/dev/null || true
  fi
  if [ -f "$work/logs/app.stdout" ]; then
    cp "$work/logs/app.stdout" "$evidence/logs/app.stdout" 2>/dev/null || true
  fi
  if [ -f "$work/logs/xvfb.log" ]; then
    cp "$work/logs/xvfb.log" "$evidence/logs/xvfb.log" 2>/dev/null || true
  fi
  rm -rf -- "$work"
}
trap cleanup EXIT

if ! command -v Xvfb >/dev/null 2>&1; then
  printf 'Xvfb is required for headless window acceptance\n' >&2
  exit 2
fi
if [ -z "$automation_bin" ] || [ ! -x "$automation_bin" ]; then
  printf 'mygo-automation is required; install it with:\n  go install github.com/egoist/mygo/plugins/automation/cmd/mygo-automation@v0.9.0\n' >&2
  exit 2
fi

{
  printf 'run_id=%s\n' "$run_id"
  printf 'commit=%s\n' "$(git -C "$root" rev-parse HEAD 2>/dev/null || printf unknown)"
  printf 'go=%s\n' "$(go version)"
  printf 'mygo=%s\n' "$(go list -m -f '{{.Version}}' github.com/egoist/mygo)"
  printf 'automation=%s\n' "$("$automation_bin" version 2>/dev/null || printf unknown)"
  printf 'kernel=%s\n' "$(uname -srm)"
  printf 'display=Xvfb\n'
  printf 'started=%s\n' "$(date -Is)"
} > "$evidence/environment.txt"

n=99
while [ -S "/tmp/.X11-unix/X$n" ]; do
  n=$((n + 1))
done
display=":$n"
Xvfb "$display" -screen 0 1440x900x24 -nolisten tcp > "$work/logs/xvfb.log" 2>&1 &
xvfb_pid=$!
for _ in $(seq 1 100); do
  [ -S "/tmp/.X11-unix/X$n" ] && break
  kill -0 "$xvfb_pid" 2>/dev/null || { printf 'Xvfb exited before display %s was ready\n' "$display" >&2; exit 1; }
  sleep 0.1
done

GOTOOLCHAIN=auto go build -tags e2e -o "$work/boxd-e2e" .

HOME="$work/home" XDG_RUNTIME_DIR="$work/runtime" XDG_CACHE_HOME="$work/home/.cache" \
  GDK_BACKEND=x11 DISPLAY="$display" NO_AT_BRIDGE=1 \
  BOXD_DATA_DIR="$work/data" BOXD_CONFIG="$work/config/config.json" \
  BOXD_START_HIDDEN=0 BOXD_SKIP_KERNEL_AUTOSTART=1 \
  MYGO_AUTOMATION="$work/app.sock" "$work/boxd-e2e" > "$work/logs/app.stdout" 2>&1 &
app_pid=$!
app_socket="$work/app.sock"

for _ in $(seq 1 200); do
  [ -S "$app_socket" ] && break
  kill -0 "$app_pid" 2>/dev/null || { printf 'boxd exited before automation socket was ready\n' >&2; exit 1; }
  sleep 0.1
done
[ -S "$app_socket" ] || { printf 'automation socket did not appear\n' >&2; exit 1; }

automation open --socket "$app_socket" > "$evidence/logs/open.txt"
windows=$(automation windows)
printf '%s\n' "$windows" > "$evidence/logs/windows.txt"
grep -q '\[content\]' "$evidence/logs/windows.txt" || { printf 'no native content window found\n' >&2; exit 1; }

automation wait --timeout 15s > "$evidence/logs/idle.txt"
automation snapshot --full > "$evidence/logs/dashboard-tree.txt"
grep -q 'Kernel controls' "$evidence/logs/dashboard-tree.txt" || { printf 'dashboard heading is missing\n' >&2; exit 1; }
automation screenshot "$evidence/screenshots/dashboard.png" > "$evidence/logs/dashboard-screenshot.txt"

pages=(nodes subscriptions connections logs config settings dashboard)
texts=("No nodes imported yet" "No subscriptions" "No active connections" "Recent activity" "Raw configuration" "Appearance" "Kernel controls")
for i in "${!pages[@]}"; do
  page=${pages[$i]}
  text=${texts[$i]}
  automation press "key=\"/$page\"" --settle --timeout 15s > "$evidence/logs/nav-$page.txt"
  automation wait --text "$text" --settle --timeout 15s > "$evidence/logs/wait-$page.txt"
  automation screenshot "$evidence/screenshots/$page.png" > "$evidence/logs/screenshot-$page.txt"
  [ -s "$evidence/screenshots/$page.png" ] || { printf 'screenshot %s is empty\n' "$page" >&2; exit 1; }
done

kill -0 "$app_pid"
automation close >/dev/null 2>&1 || true
printf 'headless acceptance passed: %s\n' "$evidence"
