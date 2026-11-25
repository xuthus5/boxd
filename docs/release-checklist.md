# Release Checklist

This checklist is the release gate for a commercial-quality build. A release is
not approved until every required item is complete.

## 1. Scope and versioning

- Confirm all changes are within the documented control-plane scope.
- Confirm backup/restore, Web Shell, and mobile/WebView host integration remain
  excluded.
- Review `CHANGELOG.md` and choose the Semantic Versioning tag.
- Confirm the engine version policy in `docs/sing-box-versioning.md`.

## 2. Automated gates

- `make fmt-check` passes.
- `make vet` passes.
- `make test` passes.
- `make race` passes.
- `govulncheck` reports no reachable Go vulnerabilities.
- `npm ci` completes from the committed lockfile.
- `npm audit --audit-level=high` reports no high or critical issues.
- `npm run check` passes, including JavaScript lint, OpenAPI validation, and
  API route coverage.
- All Playwright projects pass: desktop Chromium, Firefox, WebKit, and mobile
  Chromium.
- The accessibility scan reports no serious or critical violations.
- Container build succeeds for `linux/amd64` and `linux/arm64`.
- Native desktop cross-builds succeed for Linux, macOS and Windows, and the
  Linux desktop window passes the Xvfb/MyGo automation smoke check.

## 3. Manual product acceptance

- Initialize a fresh data directory and sign in with a strong password.
- Verify overview, inbounds, egress, routes, subscriptions, rule sets,
  observability, logs, and settings at desktop and mobile widths.
- Import a node, edit it, test it, and delete it.
- Create, edit, disable, refresh, and delete a subscription.
- Create a group and a multi-hop chain; confirm ordering and deletion.
- Change DNS and routing settings and confirm persistence after restart.
- Start, restart, and stop the kernel; verify rollback on an invalid config.
- Verify SSE logs, traffic, connection snapshots, and reconnect behavior.
- Verify invalid credentials are rate limited and no sensitive values appear
  in responses or logs.
- Verify the native desktop opens without an HTTP listener, restores window and
  theme preferences, enforces single instance, and shows no Web login controls.

## 4. Deployment verification

- Verify the binary starts from a clean directory.
- Verify the systemd package installs, starts, restarts, and uninstalls.
- Verify the uninstall default preserves data and `--purge` is explicit.
- Verify the container health check, volume persistence, and TUN guidance.
- Verify upgrade from the previous release uses SQLite migrations without data
  loss.

## 5. Security and compliance

- Review dependency and container vulnerability reports.
- Confirm no credentials or production subscription URLs are present in
  screenshots, fixtures, logs, or release assets.
- Confirm CSP, same-origin write protection, session expiry, and login rate
  limiting remain active.
- Confirm `LICENSE`, `NOTICE`, `SECURITY.md`, and `CHANGELOG.md` are included.
- Confirm release checksums and the release manifest are generated and
  published.

## 6. Publishing

- Create and push the version tag from a reviewed commit.
- Wait for all release and container workflows to complete.
- Verify GitHub Release assets, checksums, manifest, and GHCR tags.
- Verify the documented installation commands against the published assets.
- Publish release notes with upgrade, rollback, and known-limit information.

## 7. Post-release

- Monitor CI and issue reports for regressions.
- Confirm the nightly workflow produces a newer build from `main`.
- Record any follow-up work in the next milestone rather than silently
  extending the released scope.
