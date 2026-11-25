# Contributing to boxd

Thank you for improving boxd. This project values small, reviewable changes and
strict separation between the public control contracts, control-plane use
cases, adapters, and the sing-box driver.

## Development prerequisites

- Go `1.27+`.
- Node.js `24+` for browser tests and documentation tooling.
- Linux for kernel integration and TUN validation.
- `with_utls,with_quic` build tags for production-compatible builds.

## Local checks

```bash
make fmt-check
make vet
make test
make race
make build
npm ci
npm run check
npm run test:e2e:chromium
```

Install Playwright browsers once if they are not already present:

```bash
npx playwright install --with-deps chromium firefox webkit
```

## Architecture rules

- Public contracts belong in `control` and `kernel` and must not depend on
  `internal` packages.
- Business use cases belong in `internal/controlplane`.
- External systems are isolated in outbound adapters.
- Only the sing-box driver may depend on sing-box implementation packages.
- HTTP handlers perform protocol conversion, authentication, validation, and
  view redaction. They must not duplicate control-plane business rules.
- All mutations must preserve the atomic desired-state and configuration apply
  model. Do not add a second persistence path.
- Do not add Web Shell, arbitrary command execution, backup/restore, or
  upper-layer desktop/mobile host integration to the current scope.

## Before opening a pull request

- Add or update tests for behavior changes.
- Update `CHANGELOG.md` for user-visible changes.
- Update `docs/openapi.yaml` when an API route or payload changes.
- Run `npm run check:openapi` and `npm run test:e2e:chromium` for Web changes.
- Keep secrets, private subscription URLs, credentials, and production data out
  of commits, fixtures, screenshots, and issue reports.
- Describe the user impact, security implications, and verification performed.

## Commit messages

Use a concise conventional prefix such as `feat:`, `fix:`, `docs:`, `test:`,
`refactor:`, `build:`, or `chore:`. Keep unrelated changes in separate commits.

## Security

Follow [SECURITY.md](SECURITY.md) for vulnerability reports. Do not disclose
unfixed security issues in public pull requests.
