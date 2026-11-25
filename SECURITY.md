# Security Policy

## Supported versions

boxd is currently in the pre-1.0 release line. Security fixes are applied to
the latest published release and the `main` branch. Older pre-release builds
are not separately maintained after a newer release is available.

| Version | Supported |
| --- | --- |
| Latest release | Yes |
| `main` | Yes, development line |
| Older pre-1.0 releases | No |

## Reporting a vulnerability

Do not disclose security issues in a public issue, discussion, pull request, or
social media post.

Use GitHub private vulnerability reporting for the canonical repository:

<https://github.com/xuthus5/boxd/security/advisories/new>

If private reporting is unavailable, contact the repository owner through a
private channel listed on the GitHub profile and include only the information
needed to establish a secure follow-up channel.

Please include:

- Affected version or commit.
- Deployment model: binary, systemd, or container.
- Reproduction steps or a minimal proof of concept.
- Security impact and required preconditions.
- Any known mitigations.

Never include real subscription tokens, node credentials, administrator
passwords, session cookies, private keys, or production database files.

## Response process

Maintainers will make a best-effort effort to acknowledge a complete report,
validate the issue, and agree on a disclosure timeline. Timelines depend on
severity, exploitability, and upgrade complexity. Please allow time for a
coordinated fix and release before public disclosure.

When a fix is available, the project will publish a patched build, document
the impact in `CHANGELOG.md`, and credit the reporter if requested.

## Security model

boxd is a single-host control plane with one local administrator account. It
is not a multi-tenant authorization boundary.

- The HTTP API and Web console share one origin.
- Administrator authentication is a Web-ingress capability implemented in
  `internal/webauth` and `internal/adapters/inbound/httpapi`. It is not part of
  `control.Application`, so the native desktop host does not inherit Web
  passwords or browser sessions.
- The native desktop host runs the control plane in-process. It does not open
  the Web management listener and does not import `internal/webauth` or the HTTP
  inbound adapter.
- The first administrator password is initialized locally and stored as a
  salted PBKDF2-HMAC-SHA256 verifier.
- The settings console can rotate the administrator password. Rotation uses a
  server-generated 32-character random secret, accepts no user-supplied
  replacement, revokes every prior session, and displays the new secret once.
- Sessions are short-lived, in-memory bearer tokens delivered as
  `HttpOnly; SameSite=Strict` cookies to browser clients.
- Direct HTTPS is supported with `-tls-cert` and `-tls-key`. TLS 1.2 is the
  minimum version; HTTPS cookies are marked `Secure` and responses include
  HSTS.
- Cookie-authenticated writes require a same-origin request. Automated clients
  may use a bearer token returned by login.
- Login and initialization attempts are rate limited per client address.
- The service emits a restrictive Content Security Policy, disables framing,
  and does not expose a Web Shell, arbitrary command execution, backup, or
  restore API.
- node credentials and subscription query values are omitted from list views;
  full node options are available only through the authenticated node editor.

## Secure deployment requirements

- Do not expose ports `7662`, `1080`, or TUN management to untrusted networks.
- Enable native HTTPS with a trusted certificate, or terminate TLS at a trusted
  reverse proxy, VPN, or private overlay network.
- Use a strong, unique administrator password.
- Rotate the administrator password from the Web console immediately if a
  session, browser profile, or credential may have been exposed. Rotation
  invalidates all existing sessions.
- Run as a dedicated host with systemd or an equivalent process supervisor.
- Keep `/var/lib/boxd` readable only by the service account.
- Keep boxd, sing-box, the host kernel, and the reverse proxy patched.
- Restrict inbound access with a firewall or network policy.
- Review audit and application logs for unexpected authentication or
  configuration changes.
- Do not enable TUN or elevated network capabilities unless they are needed.

The container image runs as root because TUN and route management require it.
Run the container only on trusted hosts and avoid granting `NET_ADMIN` to
untrusted workloads.

## Out of scope

- Vulnerabilities in sing-box or other upstream dependencies that have not been
  shown to affect boxd's integration.
- Attacks requiring root access to the boxd host.
- Denial of service caused solely by an unauthenticated peer saturating the
  host network.
- Misconfiguration of an external reverse proxy, firewall, VPN, or container
  runtime.
- Issues that require a user to disclose credentials through an unrelated
  system.
