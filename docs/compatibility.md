# Compatibility and Support Matrix

## Server

| Item | Supported baseline |
| --- | --- |
| Operating system | Linux with a current kernel |
| Architectures | `amd64`, `arm64` |
| Deployment | Binary, systemd, Docker/OCI |
| Go build tag set | `with_utls,with_quic` |
| Engine | sing-box `v1.14.x`, locked module `v1.14.4` |
| Storage | Embedded SQLite with WAL |

## Native desktop

The desktop client is rendered with MyGo native controls and does not embed the
Web console in a WebView. It links the control plane in-process, so it does not
open the Web management port and does not require the Web administrator login.

| Platform | Architecture | Verification |
| --- | --- | --- |
| Linux | `amd64`, `arm64` | `amd64` real-window validation under Xvfb; both targets cross-compile in CI |
| macOS | `amd64`, `arm64` | Cross-compiled in CI; signing and notarization require a macOS release host |
| Windows | `amd64` | Cross-compiled in CI |

Linux TUN use still requires `/dev/net/tun` and `CAP_NET_ADMIN`. The desktop
shell itself, subscription management, node editing, routing, DNS, rule sets,
logs, connections and Mixed inbound management do not require those privileges.
Window state and the light/dark theme are persisted per user; the Web and desktop
hosts use separate default data directories to avoid SQLite writer contention.

## Browsers

The Web console is tested with the Playwright browser engines pinned in
`package-lock.json`. The supported compatibility target is the latest stable
Chromium, Firefox, and WebKit/Safari release plus the preceding major release.

| Engine | Desktop | Mobile-sized viewport |
| --- | --- | --- |
| Chromium | Supported and CI-tested | Supported and CI-tested |
| Firefox | Supported and CI-tested | Layout-tested |
| WebKit | Supported and CI-tested | Layout-tested |

Unsupported browser-specific extensions are not required. JavaScript modules,
Fetch, EventSource/SSE, CSS custom properties, and modern layout primitives are
required.

## Accessibility

The console includes semantic landmarks, keyboard-reachable controls, visible
focus states, ARIA labels for icon actions, reduced-motion handling, and
accessible dialogs and live regions. Chromium CI runs an axe-core scan and
fails on serious or critical violations. Accessibility is continuously
improving; report blockers through a normal issue with reproduction details.

## Network and TLS

boxd serves HTTP and SSE on port `7662` by default. Native HTTPS is available by
passing `-tls-cert` and `-tls-key`; TLS 1.2 is the minimum accepted version and
HTTPS responses include HSTS. A reverse proxy, VPN, or trusted private network
may still terminate TLS instead. Any proxy must preserve streaming responses,
`Host`, and forwarded headers for SSE.

## Mobile

The Web console supports current mobile browsers at a minimum 360 CSS-pixel
viewport. The sidebar becomes an off-canvas navigation panel, dialogs remain
within the viewport, and pages must not create horizontal page-level overflow.
