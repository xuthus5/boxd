#!/usr/bin/env bash
set -euo pipefail
umask 077

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"
run_id=${MYGO_ACCEPTANCE_RUN_ID:-$(date +%Y%m%d-%H%M%S)-desktop}
evidence=${MYGO_ACCEPTANCE_DIR:-$root/artifacts/acceptance}/$run_id
work=$(mktemp -d "${TMPDIR:-/tmp}/boxd-mygo-desktop.XXXXXX")
duration=${MYGO_DESKTOP_SOAK_SECONDS:-1800}
interval=${MYGO_DESKTOP_SOAK_INTERVAL:-60}
hold_seconds=${MYGO_DESKTOP_HOLD_SECONDS:-300}
automation_bin=${MYGO_AUTOMATION_BIN:-$(command -v mygo-automation || true)}
session=$(printf '%s' "boxd-desktop-$run_id" | tr -c 'A-Za-z0-9_.-' '-')

automation() {
  "$automation_bin" "$@" --session "$session"
}
app_pid=
usage() {
  cat <<'EOF'
Usage: ./scripts/mygo-desktop-acceptance.sh [--check]

Runs the automated portion of TS-026 on the current desktop display using an
isolated data directory, route checks, GPU/session capture and a soak.
Set MYGO_DESKTOP_SOAK_SECONDS, MYGO_DESKTOP_SOAK_INTERVAL or
MYGO_DESKTOP_HOLD_SECONDS to override the 1800s/60s/300s defaults.
The manual checklist remains mandatory; Xvfb is not TS-026 evidence.
EOF
}
check_number() {
  local name=$1 value=$2
  if ! [[ "$value" =~ ^[0-9]+$ ]] || [ "$value" -le 0 ]; then
    printf '%s must be a positive integer\n' "$name" >&2
    return 2
  fi
}
check_deps() {
  local missing=0
  for command in go pstree; do
    command -v "$command" >/dev/null 2>&1 || { printf 'missing required command: %s\n' "$command" >&2; missing=1; }
  done
  if [ -z "$automation_bin" ] || [ ! -x "$automation_bin" ]; then
    printf 'mygo-automation is required; install it with:\n' >&2
    printf '  go install github.com/egoist/mygo/plugins/automation/cmd/mygo-automation@v0.9.0\n' >&2
    missing=1
  fi
  [ -n "${DISPLAY:-}" ] || { printf 'DISPLAY is empty; run this inside a graphical session\n' >&2; missing=1; }
  check_number MYGO_DESKTOP_SOAK_SECONDS "$duration" || missing=1
  check_number MYGO_DESKTOP_SOAK_INTERVAL "$interval" || missing=1
  [[ "$hold_seconds" =~ ^[0-9]+$ ]] || { printf 'MYGO_DESKTOP_HOLD_SECONDS must be non-negative\n' >&2; missing=1; }
  [ "$interval" -le "$duration" ] || { printf 'soak interval must not exceed duration\n' >&2; missing=1; }
  [ "$missing" -eq 0 ] || return 2
}
cleanup() {
  if [ -n "${automation_bin:-}" ] && [ -x "$automation_bin" ]; then
    automation close >/dev/null 2>&1 || true
  fi
  if [ -n "$app_pid" ] && kill -0 "$app_pid" 2>/dev/null; then
    kill "$app_pid" 2>/dev/null || true
    wait "$app_pid" 2>/dev/null || true
  fi
  if [ -f "$work/logs/app.stdout" ]; then
    cp "$work/logs/app.stdout" "$evidence/logs/app.stdout" 2>/dev/null || true
  fi
  rm -rf -- "$work" || true
}
on_exit() {
  status=$?
  cleanup
  if [ -d "$evidence" ]; then
    printf 'finished=%s\nexit=%d\n' "$(date -Is)" "$status" >> "$evidence/commands.txt" 2>/dev/null || true
    if [ "$status" -ne 0 ] && [ ! -f "$evidence/result.md" ]; then
      printf '# Real desktop acceptance\n\n- Status: failed (exit %d)\n' "$status" > "$evidence/result.md" 2>/dev/null || true
    fi
  fi
}
trap on_exit EXIT
if [ "${1:-}" = "--check" ]; then
  check_deps
  printf 'real desktop automation dependencies are available (DISPLAY=%s)\n' "$DISPLAY"
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
if [ -n "$(git -C "$root" status --porcelain)" ]; then
  printf 'commit or clean working tree changes before generating desktop evidence\n' >&2
  exit 2
fi

display_kind=real
if command -v xdpyinfo >/dev/null 2>&1 && xdpyinfo -display "$DISPLAY" 2>/dev/null | grep -qi 'Xvfb'; then
  display_kind=xvfb
fi
if [ "$display_kind" = xvfb ] && [ "${MYGO_DESKTOP_ALLOW_XVFB:-0}" != 1 ]; then
  printf 'DISPLAY is Xvfb; set MYGO_DESKTOP_ALLOW_XVFB=1 only for script self-testing\n' >&2
  exit 2
fi
if [ "$display_kind" = xvfb ]; then
  printf 'WARNING: Xvfb self-test is not TS-026 evidence\n' >&2
fi

mkdir -p "$work/home" "$work/data" "$work/config" "$work/runtime" "$work/logs" "$evidence/logs" "$evidence/screenshots"
chmod 700 "$work" "$evidence"
printf 'command=%s\nstarted=%s\n' "$0 $*" "$(date -Is)" > "$evidence/commands.txt"

session_type=${XDG_SESSION_TYPE:-unknown}
current_desktop=${XDG_CURRENT_DESKTOP:-unknown}
gdk_backend=${GDK_BACKEND:-default}
scale=${GDK_SCALE:-1}
{
  printf 'run_id=%s\n' "$run_id"
  printf 'commit=%s\n' "$(git rev-parse HEAD 2>/dev/null || printf unknown)"
  printf 'go=%s\n' "$(go version)"
  printf 'mygo=%s\n' "$(go list -m -f '{{.Version}}' github.com/egoist/mygo)"
  printf 'automation=%s\n' "$("$automation_bin" version 2>/dev/null || printf unknown)"
  printf 'kernel=%s\n' "$(uname -srm)"
  printf 'display=%s\n' "$DISPLAY"
  printf 'display_kind=%s\n' "$display_kind"
  printf 'session_type=%s\n' "$session_type"
  printf 'current_desktop=%s\n' "$current_desktop"
  printf 'gdk_backend=%s\n' "$gdk_backend"
  printf 'gdk_scale=%s\n' "$scale"
  printf 'duration_seconds=%s\n' "$duration"
  printf 'interval_seconds=%s\n' "$interval"
  printf 'started=%s\n' "$(date -Is)"
} > "$evidence/environment.txt"

{
  printf '== displays ==\n'
  if command -v xrandr >/dev/null 2>&1; then xrandr --query 2>&1 || true; else printf 'xrandr not available\n'; fi
  printf '\n== gpu ==\n'
  if command -v glxinfo >/dev/null 2>&1; then glxinfo -B 2>&1 | sed -n '1,40p' || true; else printf 'glxinfo not available\n'; fi
  printf '\n== wayland/session ==\n'
  printf 'WAYLAND_DISPLAY=%s\n' "${WAYLAND_DISPLAY:-}"
  printf 'XMODIFIERS=%s\n' "${XMODIFIERS:-}"
  printf '\n== scaling ==\n'
  if command -v gsettings >/dev/null 2>&1; then gsettings get org.gnome.desktop.interface scaling-factor 2>&1 || true; else printf 'gsettings not available\n'; fi
  printf '\n== input methods ==\n'
  if command -v fcitx5-remote >/dev/null 2>&1; then fcitx5-remote -n 2>&1 || true; else printf 'fcitx5 not available\n'; fi
  if command -v ibus >/dev/null 2>&1; then ibus version 2>&1 | sed -n '1,5p' || true; else printf 'ibus not available\n'; fi
} > "$evidence/logs/desktop-environment.txt"

cat > "$evidence/manual-checklist.md" <<'EOF'
# Real desktop manual acceptance

Complete this checklist during the manual hold. Record issues and the Wails
comparison below before treating TS-026 as passed.

- [ ] Dashboard charts and Nodes/Subscriptions/Connections lists stay smooth.
- [ ] Logs append without forcing the view back to the bottom after scrolling up.
- [ ] Window resize, theme/page switching, tray restore and minimize work.
- [ ] Input method entry, HiDPI scaling and multi-monitor movement remain usable.
- [ ] No visible freeze, blank window, flicker or runaway CPU during 30 minutes.
- [ ] Record subjective smoothness and the same-scenario Wails comparison.

Automated metrics do not replace this checklist.

Overall result: pending
Wails comparison: pending
Issues:
EOF

printf 'build=GOTOOLCHAIN=auto go build -tags e2e -o boxd-desktop .\n' >> "$evidence/commands.txt"
GOTOOLCHAIN=auto go build -tags e2e -o "$work/boxd-desktop" .

HOME="$work/home" XDG_RUNTIME_DIR="$work/runtime" XDG_CACHE_HOME="$work/home/.cache" \
  DISPLAY="$DISPLAY" GDK_BACKEND=x11 NO_AT_BRIDGE=1 \
  BOXD_DATA_DIR="$work/data" BOXD_CONFIG="$work/config/config.json" \
  BOXD_START_HIDDEN=0 BOXD_SKIP_KERNEL_AUTOSTART=1 \
  MYGO_AUTOMATION="$work/app.sock" "$work/boxd-desktop" > "$work/logs/app.stdout" 2>&1 &
app_pid=$!
for _ in $(seq 1 200); do
  [ -S "$work/app.sock" ] && break
  kill -0 "$app_pid" 2>/dev/null || { printf 'boxd exited before automation socket was ready\n' >&2; exit 1; }
  sleep 0.1
done
[ -S "$work/app.sock" ] || { printf 'automation socket did not appear\n' >&2; exit 1; }

pstree -ap "$app_pid" > "$evidence/logs/process-tree.txt"
if grep -Eqi 'webkit|webview|wails' "$evidence/logs/process-tree.txt"; then
  printf 'webview or Wails process found in the native app process tree\n' >&2
  exit 1
fi

automation open --socket "$work/app.sock" > "$evidence/logs/open.txt"
automation windows > "$evidence/logs/windows.txt"
grep -q '\[content\]' "$evidence/logs/windows.txt" || { printf 'no native content window found\n' >&2; exit 1; }
automation wait --timeout 15s > "$evidence/logs/idle.txt"

pages=(nodes subscriptions connections logs config terminal settings dashboard)
texts=("No nodes imported yet" "No subscriptions" "No active connections" "Recent activity" "Raw configuration" "Proxy environment" "Appearance" "Kernel controls")
for i in "${!pages[@]}"; do
  page=${pages[$i]}
  text=${texts[$i]}
  automation press "key=\"/$page\"" --settle --timeout 15s > "$evidence/logs/nav-$page.txt"
  automation wait --text "$text" --settle --timeout 15s > "$evidence/logs/wait-$page.txt"
  automation screenshot "$evidence/screenshots/$page.png" > "$evidence/logs/screenshot-$page.txt"
  [ -s "$evidence/screenshots/$page.png" ] || { printf 'screenshot %s is empty\n' "$page" >&2; exit 1; }
done

printf 'timestamp,elapsed_s,rss_kb,cpu_percent,automation_ms,page,status\n' > "$evidence/soak.csv"
start=$(date +%s)
last_ticks=$(awk '{print $14 + $15}' "/proc/$app_pid/stat")
last_time=$start
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
if [ "$hold_seconds" -gt 0 ]; then
  printf 'Automated sample complete. Manually inspect the window for %s seconds; press Ctrl-C to stop early.\n' "$hold_seconds"
  sleep "$hold_seconds"
fi

samples=$(awk 'END { print (NR > 1 ? NR - 1 : 0) }' "$evidence/soak.csv")
ok_samples=$(awk -F, 'NR > 1 && $7 == "ok" { n++ } END { print n + 0 }' "$evidence/soak.csv")
min_samples=$((duration / interval))
[ "$min_samples" -gt 0 ] || min_samples=1
p95=$(awk -F, 'NR > 1 && $7 == "ok" { print $5 }' "$evidence/soak.csv" | sort -n | awk '{ a[NR]=$1 } END { idx=int((NR*95+99)/100); print (NR ? a[idx] : 0) }')
base_rss=$(awk -F, 'NR > 1 && $2 >= 300 { print $3; exit }' "$evidence/soak.csv")
[ -n "$base_rss" ] || base_rss=$(awk -F, 'NR == 2 { print $3 }' "$evidence/soak.csv")
final_rss=$(awk -F, 'NR > 1 { value = $3 } END { print value }' "$evidence/soak.csv")
if [ -z "$base_rss" ] || [ -z "$final_rss" ]; then
  printf 'soak sampling did not produce usable RSS data\n' >&2
  exit 1
fi
gpu_hint=$(glxinfo -B 2>/dev/null | awk '/OpenGL renderer string/ { print $0; exit }' || true)
[ -n "$gpu_hint" ] || gpu_hint="unknown"
avg_cpu=$(awk -F, 'NR > 1 && $7 == "ok" { sum += $4; n++ } END { if (n > 0) printf "%.2f", sum / n; else print "0.00" }' "$evidence/soak.csv")
growth=$((final_rss - base_rss))
status=passed
reason="all samples responsive; RSS and automation latency are within thresholds"
if [ "$samples" -lt "$min_samples" ]; then status=failed; reason="only $samples samples; expected at least $min_samples";
elif [ "$ok_samples" -ne "$samples" ]; then status=failed; reason="$((samples - ok_samples)) sample(s) failed";
elif [ "$p95" -gt 2000 ]; then status=failed; reason="automation p95 ${p95}ms exceeds 2000ms";
elif [ "$growth" -gt 65536 ]; then status=failed; reason="RSS grew ${growth}KiB, exceeding 65536KiB"; fi

cat > "$evidence/result.md" <<EOF
# Real desktop acceptance

- Automated status: $status
- Reason: $reason
- Commit: $(git rev-parse HEAD)
- Display kind: $display_kind
- Session type: $session_type
- Desktop: $current_desktop
- Samples: $ok_samples/$samples responsive
- Duration: ${duration}s at ${interval}s interval
- Average CPU: ${avg_cpu}%
- Baseline RSS: ${base_rss}KiB
- Final RSS: ${final_rss}KiB
- RSS growth: ${growth}KiB (threshold 65536KiB)
- Automation p95: ${p95}ms (threshold 2000ms)
- GPU: ${gpu_hint}
- Evidence: environment.txt, soak.csv, logs/, screenshots/

Manual result: pending in manual-checklist.md.
Only a non-Xvfb run with a completed manual checklist can satisfy TS-026.
EOF

if [ "$display_kind" = xvfb ]; then
  printf 'self-test passed under Xvfb; this does not satisfy TS-026\n' >&2
fi
if [ "$status" != passed ]; then
  printf 'desktop acceptance failed: %s (%s)\n' "$reason" "$evidence" >&2
  exit 1
fi
printf 'desktop automated acceptance passed: %s\n' "$evidence"
printf 'Complete %s/manual-checklist.md before treating TS-026 as passed.\n' "$evidence"
