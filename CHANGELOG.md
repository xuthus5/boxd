# Changelog

All notable changes to boxd are documented here. The project follows
[Semantic Versioning](https://semver.org/) after the first stable release and
uses [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) conventions.

## [Unreleased]

### Added

- Embedded responsive Web control plane covering overview, inbounds, egress
  nodes, groups, chains, routing, DNS, subscriptions, rule sets,
  observability, streaming application/kernel logs, and settings.
- Real REST and SSE APIs for kernel lifecycle, atomic configuration apply,
  subscription synchronization, node import and latency tests, DNS probes,
  connection snapshots, and runtime logs.
- SQLite-backed desired state, configuration history, node test history,
  settings, credentials, and audit events.
- sing-box v1.14.x driver with transaction rollback, runtime metrics, and
  support for VLESS Reality and QUIC-based protocols.
- Linux `amd64` and `arm64` binaries, hardened systemd unit, installation and
  uninstallation scripts, and multi-architecture GHCR images.
- GitHub Actions for CI, release assets, nightly builds, and container
  publishing.
- Playwright end-to-end coverage for authentication, navigation, CRUD,
  responsive layouts, cross-browser behavior, accessibility, and kernel
  lifecycle, executed in CI.
- OpenAPI 3.1 contract, deployment documentation, browser compatibility
  matrix, security policy, and release checklist.
- Apache-2.0 licensing, NOTICE attribution, dependency update automation, and
  release packages that include governance and API documentation.

### Security

- PBKDF2 password verifiers, rate-limited authentication, in-memory sessions,
  same-origin write protection, strict CSP, and sensitive-view redaction.
- Go module and standard-library dependency baselines are checked with
  `govulncheck`; known reachable vulnerabilities block release.

### Known limitations

- Single local administrator only; multi-user RBAC, SSO, and multi-tenancy are
  not part of the current product boundary.
- Backup and restore are intentionally excluded from the product.
- Desktop and mobile host integrations are deferred until the control-plane,
  kernel, and model contracts are stable.
