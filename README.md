# boxd

[English](README.md) | [简体中文](README.zh-CN.md)

Single-node control plane for [sing-box](https://github.com/SagerNet/sing-box). Manage kernel config, subscriptions, route/DNS policy, and runtime observability from a web panel.

> PoC stage — APIs and data models may change.

## Features

- **Dashboard**: start/stop/restart kernel, live up/down traffic, memory, recent logs
- **Proxy config**: structured inbound/outbound forms (TLS/Reality, transports), plus Advanced JSON
- **Traffic policy**: route rules and rule-sets, DNS servers and rules; rule name/description stored by boxd
- **Nodes & subscriptions**: subscription group cards, imported singles, per-node and batch TCP/HTTP/ICMP probes
- **Observability**: kernel logs, application logs, active connections
- **Advanced**: Endpoints, Experimental, full kernel JSON
- **Settings**: theme, language, password/JWT rotation, probe URLs, global URLTest defaults, log level

## Stack

| Layer | Choice |
| --- | --- |
| Backend | Go, chi, bbolt, statically linked sing-box |
| Frontend | React 19, TypeScript, Vite, shadcn/ui, Tailwind CSS |
| Auth | JWT (HS256), Argon2id passwords |

## Quick start

### Requirements

- Go 1.26+
- Node.js 20+
- Linux recommended for production

### Build and run

```bash
make build
BOXD_PASSWORD='your-strong-password' \
  BOXD_DATA_DIR=./data \
  BOXD_CONFIG=./data/config.json \
  ./bin/boxd
```

Open `http://127.0.0.1:9091`. Default username is `admin`. On the default password, the UI forces a password change.

### Dev mode (split frontend/backend)

```bash
# Terminal 1: Vite UI (default http://127.0.0.1:5173)
cd ui && npm ci && npm run dev

# Terminal 2: API (default [::]:9091)
export BOXD_PASSWORD='dev-password'
export BOXD_DATA_DIR="$PWD/data"
export BOXD_CONFIG="$PWD/data/config.json"
export BOXD_CORS_ALLOWED_ORIGINS='http://127.0.0.1:5173,http://localhost:5173'
go run ./cmd/boxd/
```

`make dev` starts the UI in the background and runs the backend for a quick smoke run.

## Usage

1. Sign in and rotate the admin password.
2. **Subscriptions / nodes**: add a subscription URL or import a single node; configure URLTest (inherit global defaults when needed).
3. **Inbounds / outbounds**: create local proxy inbounds; bind subscription groups as selector/urltest, or use direct/block.
4. **Route / DNS**: edit rules in forms; install common default rules and rule-sets when useful.
5. **Dashboard**: start the kernel; watch traffic and logs.
6. **Settings**: theme, language, minimum log level, system probe URLs, kernel autostart.

### Built-in routing helpers

The route page can install common rules (sniff, hijack DNS, bypass LAN/ICMP, block QUIC/ads, CN domain/IP split, etc.). Rule-sets include Loyalsoldier text sets (local convert) and SagerNet binary sets (remote cache).

### Backup and restore

```bash
./bin/boxd --backup /var/backups/boxd/boxd-$(date +%F).tar.gz
systemctl stop boxd.service   # if running as a service
./bin/boxd --restore /var/backups/boxd/boxd-YYYY-MM-DD.tar.gz \
  --data-dir /var/lib/boxd --config /etc/sing-box/config.json
systemctl start boxd.service
```

The archive database entry is `boxd.db`.

## Configuration

| Env | Flag | Default | Description |
| --- | --- | --- | --- |
| `BOXD_LISTEN` | `--listen` | `[::]:9091` | Listen address (wins over `BOXD_PORT`) |
| `BOXD_PORT` | - | `9091` | Port-only form |
| `BOXD_CONFIG` | `--config` | `/etc/sing-box/config.json` | Kernel config path |
| `BOXD_DATA_DIR` | `--data-dir` | `/var/lib/boxd` | Data dir (DB, backups, rule-set cache) |
| `BOXD_USERNAME` | `--username` | `admin` | Login username |
| `BOXD_PASSWORD` | `--password` | `admin123` | First-run password only; ignored once a hash exists |
| `BOXD_LOG_LEVEL` | `--log-level` | `info` | Application log level |
| `BOXD_REFRESH_INTERVAL` | `--refresh-interval` | `60` | Subscription refresh interval (minutes) |
| `BOXD_TLS_CERT` | `--tls-cert` | - | TLS certificate path |
| `BOXD_TLS_KEY` | `--tls-key` | - | TLS private key path |
| `BOXD_CORS_ALLOWED_ORIGINS` | - | - | Comma-separated CORS origins |

### Auth notes

- Passwords: Argon2id in bbolt. Priority: stored hash → first-run `BOXD_PASSWORD` → `admin123`.
- JWT: secret generated on first start and stored in the DB; rotation from Settings invalidates all sessions.
- Default password keeps a persistent warning and forces rotation.

## Install from release archive

Nightly and formal release archives are Linux amd64 tarballs. Typical contents:

- `boxd` — binary
- `boxd.service` — systemd unit
- `boxd.env.example` — optional environment template (included when present in the repo)
- docs / licenses

boxd itself does **not** auto-load a `.env` file. Configuration comes from:

1. process environment variables (`BOXD_*`)
2. CLI flags
3. when run under systemd, `EnvironmentFile=-/etc/boxd/boxd.env` in `boxd.service` (the leading `-` means “optional if missing”)

### systemd install (recommended)

```bash
# 1. Unpack
tar -xzf boxd_v0.1.0_linux_amd64.tar.gz -C /tmp/boxd-release
cd /tmp/boxd-release

# 2. System user and directories
useradd --system --home /var/lib/boxd --shell /sbin/nologin boxd || true
install -d -o boxd -g boxd -m 0700 /var/lib/boxd
install -d -o root -g boxd -m 0750 /etc/boxd
install -d -o boxd -g boxd -m 0750 /etc/sing-box

# 3. Binary + unit + env
install -o root -g boxd -m 0750 ./boxd /usr/local/bin/boxd
install -m 0644 ./boxd.service /etc/systemd/system/boxd.service
if [[ -f ./boxd.env.example ]]; then
  install -o root -g boxd -m 0640 ./boxd.env.example /etc/boxd/boxd.env
  # set BOXD_PASSWORD and review other values
  ${EDITOR:-vi} /etc/boxd/boxd.env
else
  printf '%s\n' 'BOXD_PASSWORD=your-strong-password' > /etc/boxd/boxd.env
  chown root:boxd /etc/boxd/boxd.env
  chmod 0640 /etc/boxd/boxd.env
fi

# 4. Start
systemctl daemon-reload
systemctl enable --now boxd.service
systemctl status boxd.service
```

Open `http://127.0.0.1:9091` (or your listen address). Default username is `admin`.

### Run the binary without systemd

```bash
export BOXD_PASSWORD='your-strong-password'
export BOXD_DATA_DIR=/var/lib/boxd
export BOXD_CONFIG=/etc/sing-box/config.json
/usr/local/bin/boxd
```

Or pass flags:

```bash
/usr/local/bin/boxd \
  --data-dir /var/lib/boxd \
  --config /etc/sing-box/config.json \
  --password 'your-strong-password'
```

## Docker

Public images (after CI push):

```bash
# Nightly from main
docker pull ghcr.io/xuthus5/boxd:nightly

# Formal release
docker pull ghcr.io/xuthus5/boxd:latest
docker pull ghcr.io/xuthus5/boxd:v0.1.0
```

### Run with Docker

```bash
docker run -d --name boxd --restart unless-stopped \
  -p 9091:9091 \
  -e BOXD_PASSWORD='your-strong-password' \
  -e BOXD_LISTEN='[::]:9091' \
  -v boxd-data:/var/lib/boxd \
  -v boxd-config:/etc/sing-box \
  --cap-add NET_ADMIN \
  ghcr.io/xuthus5/boxd:latest
```

Notes:

- Pass config via `-e BOXD_*` (or `--env-file`); the container process does not read `/etc/boxd/boxd.env` by itself.
- Persist `/var/lib/boxd` (database, caches) and `/etc/sing-box` (kernel config).
- `NET_ADMIN` is recommended when using TUN / advanced networking features of sing-box.
- Health check hits `http://127.0.0.1:9091/healthz` inside the container.

### Build the image locally

```bash
docker build --build-arg VERSION=dev -t boxd:local .
docker run --rm -p 9091:9091 -e BOXD_PASSWORD='dev-password' boxd:local
```

## Deploy

For binary installs, prefer [Install from release archive](#install-from-release-archive).
For containers, prefer [Docker](#docker).

### TLS

Built-in TLS:

```bash
BOXD_TLS_CERT=/etc/boxd/tls/fullchain.pem \
BOXD_TLS_KEY=/etc/boxd/tls/privkey.pem \
/usr/local/bin/boxd
```

Or terminate TLS with Caddy / Nginx / Traefik, keep upstream on `127.0.0.1:9091`, and pass through WebSocket/SSE.

Release gates and rollback: [docs/boxd/release-checklist.md](docs/boxd/release-checklist.md), [docs/operations.md](docs/operations.md).


## CI artifacts and Docker images

| Trigger | Binary | Docker (GHCR) |
| --- | --- | --- |
| Pull request | quality checks only | build smoke (`push: false`) |
| Push to `main` | rolling **nightly** GitHub Release + Actions artifact (`boxd_nightly_linux_amd64.tar.gz`) | `ghcr.io/<owner>/boxd:nightly` |
| Tag `v*` | formal GitHub Release + SBOM | `ghcr.io/<owner>/boxd:<tag>`, `:<version>`, `:latest` |

Examples (replace owner):

```bash
# Nightly image
docker pull ghcr.io/xuthus5/boxd:nightly

# Stable release image
docker pull ghcr.io/xuthus5/boxd:latest
docker pull ghcr.io/xuthus5/boxd:v0.1.0
```

Nightly binary archives are published under the rolling GitHub Release tag `nightly` and also uploaded as workflow artifacts (14-day retention). Formal builds are produced by pushing a `v*` tag.

## Local development

### Day-to-day

```bash
make build              # UI build + embed + bin/boxd
make dev                # quick split run
make clean              # remove bin and dist
make check-go           # tests, race, coverage ≥90%, lint, govulncheck
make check-ui           # typecheck / lint / coverage / build
make check-embedded-ui  # embedded asset integrity
```

Frontend:

```bash
cd ui
npm ci
npm run dev
npm run check
npm run e2e:install && npm run e2e   # Playwright mocks; does not hit production :9091
```

Backend:

```bash
go run ./cmd/boxd/
go test ./...
golangci-lint run ./...
goimports-reviser -rm-unused -set-alias -project-name github.com/xuthus5/boxd -recursive ./internal
goimports-reviser -rm-unused -set-alias -project-name github.com/xuthus5/boxd -recursive ./cmd
```

### Release package

```bash
./scripts/package-release.sh v0.1.0 release
# produces release/boxd_v0.1.0_linux_amd64.tar.gz and sha256
```

Pushing a `v*` tag runs the GitHub Release workflow (full gates, archive, SBOM).

### Live smoke

```bash
BOXD_PASSWORD='your-password' ./scripts/e2e-live.sh
BOXD_PASSWORD='your-password' ./scripts/soak-runtime.sh
```

## License

boxd **first-party code** is under [Apache License 2.0](LICENSE).

Release binaries **statically link** GPL-3.0 sing-box / sing. Distributing those binaries also requires the GPL-3.0 obligations described in [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) (corresponding source and build information). This is not legal advice.
