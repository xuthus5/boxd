#!/usr/bin/env bash
set -euo pipefail
umask 077

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"
run_id=${MYGO_ACCEPTANCE_RUN_ID:-$(date +%Y%m%d-%H%M%S)-soak}
evidence=${MYGO_ACCEPTANCE_DIR:-$root/artifacts/acceptance}/$run_id
work=$(mktemp -d "${TMPDIR:-/tmp}/boxd-mygo-soak.XXXXXX")
duration=${MYGO_SOAK_SECONDS:-1800}
interval=${MYGO_SOAK_INTERVAL:-60}
automation_bin=${MYGO_AUTOMATION_BIN:-$(command -v mygo-automation || true)}
session=$(printf '%s' "boxd-soak-$run_id" | tr -c 'A-Za-z0-9_.-' '-')

automation() {
  "$automation_bin" "$@" --session "$session"
}
app_pid=
xvfb_pid=

mkdir -m 700 -p "$work/home" "$work/data" "$work/config" "$work/runtime" "$work/logs" "$evidence/logs" "$evidence/screenshots"
chmod 700 "$work" "$evidence"

cleanup() {
  if [ -n "${automation_bin:-}" ]; then automation close >/dev/null 2>&1 || true; fi
  if [ -n "$app_pid" ] && kill -0 "$app_pid" 2>/dev/null; then kill "$app_pid" 2>/dev/null || true; wait "$app_pid" 2>/dev/null || true; fi
  if [ -n "$xvfb_pid" ] && kill -0 "$xvfb_pid" 2>/dev/null; then kill "$xvfb_pid" 2>/dev/null || true; wait "$xvfb_pid" 2>/dev/null || true; fi
  for f in app.stdout xvfb.log soak.log; do
    if [ -f "$work/logs/$f" ]; then cp "$work/logs/$f" "$evidence/logs/$f" 2>/dev/null || true; fi
  done
  rm -rf -- "$work"
}
trap cleanup EXIT

if ! command -v Xvfb >/dev/null 2>&1 || [ -z "$automation_bin" ] || [ ! -x "$automation_bin" ]; then
  printf 'Xvfb and mygo-automation are required\n' >&2
  exit 2
fi

{
  printf 'run_id=%s\n' "$run_id"
  printf 'commit=%s\n' "$(git -C "$root" rev-parse HEAD 2>/dev/null || printf unknown)"
  printf 'go=%s\n' "$(go version)"
  printf 'mygo=%s\n' "$(go list -m -f '{{.Version}}' github.com/egoist/mygo)"
  printf 'duration_seconds=%s\n' "$duration"
  printf 'interval_seconds=%s\n' "$interval"
  printf 'kernel=%s\n' "$(uname -srm)"
  printf 'started=%s\n' "$(date -Is)"
} > "$evidence/environment.txt"

n=99
while [ -S "/tmp/.X11-unix/X$n" ]; do n=$((n + 1)); done
display=":$n"
Xvfb "$display" -screen 0 1440x900x24 -nolisten tcp > "$work/logs/xvfb.log" 2>&1 &
xvfb_pid=$!
for _ in $(seq 1 100); do
  [ -S "/tmp/.X11-unix/X$n" ] && break
  kill -0 "$xvfb_pid" 2>/dev/null || { printf 'Xvfb exited before display was ready\n' >&2; exit 1; }
  sleep 0.1
done

GOTOOLCHAIN=auto go build -tags e2e -o "$work/boxd-e2e" .
HOME="$work/home" XDG_RUNTIME_DIR="$work/runtime" XDG_CACHE_HOME="$work/home/.cache" \
  GDK_BACKEND=x11 DISPLAY="$display" NO_AT_BRIDGE=1 \
  BOXD_DATA_DIR="$work/data" BOXD_CONFIG="$work/config/config.json" \
  BOXD_START_HIDDEN=0 BOXD_SKIP_KERNEL_AUTOSTART=1 \
  MYGO_AUTOMATION="$work/app.sock" "$work/boxd-e2e" > "$work/logs/app.stdout" 2>&1 &
app_pid=$!
for _ in $(seq 1 200); do
  [ -S "$work/app.sock" ] && break
  kill -0 "$app_pid" 2>/dev/null || { printf 'boxd exited before automation socket was ready\n' >&2; exit 1; }
  sleep 0.1
done
[ -S "$work/app.sock" ] || { printf 'automation socket did not appear\n' >&2; exit 1; }

automation open --socket "$work/app.sock" > "$evidence/logs/open.txt"
automation wait --timeout 15s > "$evidence/logs/idle.txt"
printf 'timestamp,elapsed_s,rss_kb,cpu_percent,automation_ms,page,status\n' > "$evidence/soak.csv"

start=$(date +%s)
last_ticks=$(awk '{print $14 + $15}' "/proc/$app_pid/stat")
last_time=$start
pages=(dashboard nodes subscriptions connections logs settings)
sample=0
while :; do
  now=$(date +%s)
  elapsed=$((now - start))
  [ "$elapsed" -ge "$duration" ] && break
  sleep "$((interval < duration - elapsed ? interval : duration - elapsed))"
  now=$(date +%s)
  elapsed=$((now - start))
  page=${pages[$((sample % ${#pages[@]}))]}
  begin_ns=$(date +%s%N)
  if ! automation press "key=\"/$page\"" --settle --timeout 15s >> "$work/logs/soak.log" 2>&1; then
    printf '%s,%d,,, ,%s,fail\n' "$(date -u +%FT%TZ)" "$elapsed" "$page" >> "$evidence/soak.csv"
    printf 'soak navigation failed on page %s\n' "$page" >&2
    exit 1
  fi
  automation_ms=$((($(date +%s%N) - begin_ns) / 1000000))
  now_ticks=$(awk '{print $14 + $15}' "/proc/$app_pid/stat")
  hz=$(getconf CLK_TCK)
  cpu=$(awk -v d=$((now_ticks - last_ticks)) -v hz="$hz" -v dt=$((now - last_time)) 'BEGIN { if (dt > 0) printf "%.2f", 100 * (d / hz) / dt; else print "0.00" }')
  rss=$(awk '/VmRSS:/ {print $2}' "/proc/$app_pid/status")
  printf '%s,%d,%s,%s,%d,%s,ok\n' "$(date -u +%FT%TZ)" "$elapsed" "$rss" "$cpu" "$automation_ms" "$page" >> "$evidence/soak.csv"
  last_ticks=$now_ticks
  last_time=$now
  sample=$((sample + 1))
done

automation screenshot "$evidence/screenshots/soak-end.png" > "$evidence/logs/soak-end-screenshot.txt"
printf 'soak passed: %s\n' "$evidence"
