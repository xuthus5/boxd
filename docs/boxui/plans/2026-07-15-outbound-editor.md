# Outbound Editor Implementation Plan

> **Execution:** Implement inline in the current session. Do not use superpowers implementation skills, per the user's constraint.

**Goal:** Replace the minimal outbound editor with a type-aware sing-box 1.13.14 form while preserving complete JSON editing and unknown fields.

**Architecture:** Extract the proven generic field renderer and nested JSON helpers from the inbound editor into proxy-shared modules. Keep inbound and outbound protocol metadata, type transitions, and tab composition separate so shared rendering behavior does not couple protocol rules.

**Tech Stack:** React 19, TypeScript 6, Vite, shadcn/Base UI, Tailwind CSS v4, Vitest, Testing Library, Playwright.

## Global Constraints

- Do not modify backend code or API contracts.
- Use existing shadcn components and no new global styles.
- Preserve unknown JSON fields and valid shared fields during type changes.
- Exclude removed `dns`, `wireguard`, and `shadowsocksr` outbound types.
- Keep files at or below 300 lines and functions at or below 50 lines.
- Maintain statements, branches, functions, and lines coverage at or above 90%.

---

### Task 1: Extract shared proxy form primitives

**Files:**

- Create: `ui/src/features/proxy/proxy-form-model.ts`
- Create: `ui/src/features/proxy/proxy-form-fields.tsx`
- Modify: `ui/src/features/proxy/inbound-form-model.ts`
- Modify: `ui/src/features/proxy/inbound-form-fields.tsx`
- Test: `ui/src/features/proxy/inbound-form.test.tsx`

**Interfaces:**

- `FieldKind`, `FieldSpec`, `JsonObject`
- `getPath(object, path): JsonValue | undefined`
- `setPath(object, path, value): JsonObject`
- `FieldTransform(object, field, raw): JsonObject | null | undefined`
- `ProxyFormFields({ fields, object, onChange, onFieldValidityChange, transformField })`

- [ ] Move generic types and nested path helpers into `proxy-form-model.ts` and re-export them from the inbound model during migration.
- [ ] Move text, boolean, Select, list, numeric-list, JSON object/array rendering into `proxy-form-fields.tsx`.
- [ ] Add an optional field transform hook; `undefined` invokes default path conversion, `null` rejects invalid input, and a JSON object applies a protocol-specific transition.
- [ ] Keep `InboundFormFields` as a thin wrapper that applies `changeTransportType` only to `transport.type`.
- [ ] Run `npm --prefix ui test -- --run src/features/proxy/inbound-form.test.tsx src/features/proxy/inbound-editor.test.tsx`; expect all tests to pass without behavior changes.

### Task 2: Define outbound field metadata and transitions

**Files:**

- Create: `ui/src/features/proxy/outbound-form-model.ts`
- Test: `ui/src/features/proxy/outbound-form.test.ts`

**Interfaces:**

```ts
export const outboundTypes: readonly string[]
export const serverTypes: ReadonlySet<string>
export const dialerTypes: ReadonlySet<string>
export const outboundTLSTypes: ReadonlySet<string>
export const outboundTransportTypes: ReadonlySet<string>
export const outboundMultiplexTypes: ReadonlySet<string>
export function protocolFields(type: string): FieldSpec[]
export function groupFields(type: string): FieldSpec[]
export function transportTypeFields(type: string): FieldSpec[]
export function changeOutboundType(object: JsonObject, type: string): JsonObject
export function changeOutboundTransportType(object: JsonObject, type: string): JsonObject
```

- [ ] Add the valid type list: direct, block, selector, urltest, socks, http, shadowsocks, vmess, vless, trojan, naive, hysteria, hysteria2, tuic, ssh, tor, shadowtls, anytls.
- [ ] Define common `DialerOptions`, including detour, bind addresses, marks, timeout, TCP/UDP options, structured domain resolver, network strategy, network types, and fallback delay.
- [ ] Define protocol fields with exact sing-box JSON paths and value types; model HTTP headers and Tor options as validated JSON objects.
- [ ] Define outbound TLS, uTLS, ECH, Reality, client certificate, transport, UDP-over-TCP, and full outbound multiplex fields.
- [ ] Define selector and URLTest members, defaults, URL, interval, tolerance, idle timeout, and connection interruption.
- [ ] Implement type transitions that preserve compatible server, dialer, TLS, transport, multiplex, group, and unknown nested fields while clearing credentials and incompatible known paths.
- [ ] Implement transport transitions that preserve shared path/headers and unknown transport fields but remove subtype-specific fields with incompatible JSON types.
- [ ] Test valid/removed types, shared-field preservation, credential cleanup, group transitions, transport host type differences, and unknown nested fields.

### Task 3: Build the modular outbound editor

**Files:**

- Create: `ui/src/features/proxy/outbound-editor-dialog.tsx`
- Create: `ui/src/features/proxy/outbound-editor.test.tsx`
- Modify: `ui/src/features/proxy/proxy-editor-dialog.tsx`
- Modify: `ui/src/features/proxy/proxy-outbound-editor.test.tsx`

**Behavior:**

```text
Dialog
├── Basic
├── Dialing & network
├── Protocol / Group
├── TLS / uTLS / Reality
├── Transport & multiplex
└── Advanced JSON
```

- [ ] Write failing tests for the type Select, removed type exclusion, VLESS TLS/uTLS/Reality, transport/multiplex, Hysteria2/TUIC/SSH fields, selector/URLTest groups, invalid JSON, unknown fields, and responsive Dialog width.
- [ ] Add a `sm:max-w-5xl` scroll-safe Dialog using `Tabs`, `FieldGroup`, `Field`, `Select`, `Switch`, `Input`, `Textarea`, `Alert`, and the existing `JsonEditor`.
- [ ] Show server/port only for server-based protocols and DialerOptions only for dialer-based types.
- [ ] Keep protocol and transport panels mounted so invalid structured JSON remains visible to the Dialog validity state across tab changes.
- [ ] Display a shadcn `Alert` for selector/URLTest explaining that subscription-generated groups should be managed from Subscriptions.
- [ ] Disable save when the root is not an object, type is missing, or a structured field is invalid.
- [ ] Delegate outbound editing from `ProxyEditorDialog` to `OutboundEditorDialog`; leave inbound delegation unchanged.
- [ ] Run `npm --prefix ui test -- --run src/features/proxy`; expect all proxy tests to pass.

### Task 4: Add copy, documentation, and regression coverage

**Files:**

- Modify: `ui/src/i18n/locales/zh.ts`
- Modify: `ui/src/i18n/locales/en.ts`
- Modify: `README.md`
- Modify: `ui/src/features/proxy/proxy-interactions.test.tsx`

- [ ] Add concise Chinese and English labels for outbound tabs, DialerOptions, protocol fields, TLS/uTLS/Reality, groups, JSON hints, and subscription ownership warning.
- [ ] Update the README proxy capability line to mention structured inbound and outbound editing without expanding unrelated documentation.
- [ ] Update the outbound workflow test to use the type Select and new server labels.
- [ ] Run `npm --prefix ui run typecheck`, `npm --prefix ui run lint`, and targeted proxy tests; expect zero errors.

### Task 5: Review, verify, commit, and deploy

**Files:**

- Review all files changed by Tasks 1–4.

- [ ] Request a read-only code review focused on sing-box field names/types, type-transition data loss, shadcn composition, accessibility, and mobile layout.
- [ ] Fix every Critical and Important finding and re-run targeted tests.
- [ ] Run `make check-ui`; expect all four coverage metrics at or above 90%.
- [ ] Run `make check-go`; expect tests, race, Go coverage, golangci-lint, and govulncheck to pass.
- [ ] Run `cd ui && npm run e2e`; expect all Playwright tests to pass.
- [ ] Run `git diff --check`, then commit only task files with `feat(outbounds): expand editor fields`.
- [ ] Run `make build && make check-embedded-ui`.
- [ ] Install `bin/boxui` as `/usr/local/bin/boxui` with owner `root:boxui` and mode `0750`, restart `boxui.service`, and verify active state, executable path, matching hash, startup logs, and HTTP 200.
