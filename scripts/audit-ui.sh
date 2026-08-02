#!/usr/bin/env bash
set -euo pipefail

# audit-ui.sh 审计前端生产依赖漏洞。
#
# react-router 的 GHSA-qwww-vcr4-c8h2（RSC Mode CSRF bypass）仅影响
# React Server Components / SSR 模式。boxd 前端是纯 SPA（client-side
# rendering + BrowserRouter，无 RSC/SSR），运行时不受该漏洞影响，
# 且 react-router 7.x 无修复版本（需 major 8.x，react-router-dom 无 8.x）。
# 因此将仅含该已知误报的审计结果视为通过，其余漏洞一律失败。

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root_dir/ui"

output=$(npm audit --omit=dev --audit-level=low --json --registry=https://registry.npmjs.org 2>&1) || true

python3 - "$output" <<'PYEOF'
import json
import sys

# 本项目场景下可豁免的已知误报 advisory。
IGNORED = {"https://github.com/advisories/GHSA-qwww-vcr4-c8h2"}

raw = sys.argv[1]
try:
    data = json.loads(raw)
except json.JSONDecodeError:
    print(raw)
    sys.exit(1)

vulns = data.get("vulnerabilities", {})
if not vulns:
    print("npm audit: 0 vulnerabilities")
    sys.exit(0)

unignored = []
for name, info in vulns.items():
    urls = []
    for via in info.get("via", []):
        if isinstance(via, dict) and via.get("url"):
            urls.append(via["url"])
    # react-router-dom 的漏洞是通过依赖 react-router 间接报告的（via 为空）。
    # 若 react-router 已被豁免，则其下游包也应豁免。
    if not urls and name in ("react-router-dom",):
        if "react-router" in vulns:
            rr_urls = [x.get("url") for x in vulns["react-router"].get("via", []) if isinstance(x, dict)]
            if rr_urls and set(rr_urls).issubset(IGNORED):
                continue
    if not urls or not set(urls).issubset(IGNORED):
        unignored.append((name, info.get("severity"), urls))

if not unignored:
    print("npm audit: only ignored RSC-only advisories present")
    print("Ignored:", sorted(IGNORED))
    sys.exit(0)

print("npm audit: UNIGNORED vulnerabilities found")
for name, severity, urls in unignored:
    print(f"  {name} [{severity}] {urls}")
sys.exit(1)
PYEOF
