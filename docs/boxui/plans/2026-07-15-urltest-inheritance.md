# URLTest Inheritance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Add global sing-box URLTest defaults and optional per-subscription field overrides that are resolved whenever managed outbounds are synchronized.

**Architecture:** Store a complete `URLTestDefaults` object in the existing settings bucket and optional pointer fields in each subscription. Keep validation and merge semantics in focused core helpers, expose typed settings and subscription APIs, then compose shadcn forms that save data and invoke the existing configuration sync endpoint.

**Tech Stack:** Go, chi, bbolt, React 19, TypeScript, TanStack Query, shadcn/ui, Vitest.

## Global Constraints

- Keep the existing TCP/HTTP/ICMP test URL independent from sing-box URLTest defaults.
- Use shadcn components only; do not add global custom styles.
- Preserve old subscription JSON without migrations.
- All changed behavior must have tests and repository coverage must remain at least 90%.
- Run goimports-reviser, golangci-lint, Go tests, UI checks, production build, embedded UI verification, Git commit, and local systemd deployment.
- Do not add `idle_timeout`, `interrupt_exist_connections`, dependencies, schema migrations, or unrelated refactors.

---

### Task 1: URLTest domain model and settings persistence

**Files:**
- Modify: `internal/model/types.go`
- Modify: `internal/core/settings.go`
- Test: `internal/core/settings_test.go`

**Interfaces:**
- Produces: `model.URLTestDefaults`, `model.URLTestOverrides`, `DefaultURLTestDefaults()`, `ValidateURLTestDefaults(model.URLTestDefaults) error`, `ValidateURLTestOverrides(*model.URLTestOverrides) error`, `ResolveURLTest(model.URLTestDefaults, *model.URLTestOverrides) model.URLTestDefaults`.
- Produces: `(*SettingsManager).URLTestDefaults() (model.URLTestDefaults, error)` and `SetURLTestDefaults(model.URLTestDefaults) error`.

- [x] **Step 1: Write failing core tests**

Cover default values, JSON persistence, missing setting fallback, invalid stored JSON, URL scheme validation, positive duration validation, tolerance range, `false`/`0` pointer overrides, and partial merge behavior.

- [x] **Step 2: Run the focused test and confirm failure**

Run: `go test ./internal/core -run 'URLTest|Settings'`

Expected: compile failure because URLTest types and methods do not exist.

- [x] **Step 3: Implement the minimal domain and persistence code**

Use these shapes:

```go
type URLTestDefaults struct {
	Enabled   bool   `json:"enabled"`
	URL       string `json:"url"`
	Interval  string `json:"interval"`
	Tolerance uint16 `json:"tolerance"`
}

type URLTestOverrides struct {
	Enabled   *bool   `json:"enabled,omitempty"`
	URL       *string `json:"url,omitempty"`
	Interval  *string `json:"interval,omitempty"`
	Tolerance *uint16 `json:"tolerance,omitempty"`
}
```

Validate absolute HTTP(S) URLs with `net/url`, durations with `time.ParseDuration`, and persist JSON under `subscription_urltest_defaults`.

- [x] **Step 4: Run focused tests**

Run: `go test ./internal/core -run 'URLTest|Settings'`

Expected: PASS.

---

### Task 2: Typed global settings API

**Files:**
- Modify: `internal/api/settings_handler.go`
- Modify: `internal/api/router.go`
- Test: `internal/api/auth_config_test.go`

**Interfaces:**
- Produces: `GET /api/settings/urltest-defaults`.
- Produces: `PUT /api/settings/urltest-defaults` accepting and returning `model.URLTestDefaults`.

- [x] **Step 1: Write failing handler tests**

Verify default GET, successful PUT and reload, malformed JSON, invalid URL, zero/invalid interval, and authenticated router exposure.

- [x] **Step 2: Run focused tests and confirm failure**

Run: `go test ./internal/api -run 'URLTestDefaults|ImportAndSettings'`

Expected: FAIL because handlers and routes are missing.

- [x] **Step 3: Implement handlers and routes**

Add `GetURLTestDefaults` and `SetURLTestDefaults`. Decode once, call core validation/persistence, and return the repository's `invalid_request` or `internal` error envelope consistently.

- [x] **Step 4: Run focused tests**

Run: `go test ./internal/api -run 'URLTestDefaults|ImportAndSettings'`

Expected: PASS.

---

### Task 3: Subscription override persistence and API

**Files:**
- Modify: `internal/model/types.go`
- Modify: `internal/core/subscription.go`
- Modify: `internal/api/subscription_handler.go`
- Test: `internal/core/subscription_test.go`
- Test: `internal/api/nodes_subscription_test.go`

**Interfaces:**
- Extends: `model.Subscription.URLTest *model.URLTestOverrides` with JSON key `urltest`.
- Produces: `core.SubscriptionParams` containing `Name`, `URL`, `IntervalMin`, and `URLTest`.
- Changes: `SubscriptionManager.Create(params SubscriptionParams)`.
- Changes: `SubscriptionManager.Update(id string, params SubscriptionParams)`.
- Extends: subscription POST/PUT request bodies with nullable `urltest`.

- [x] **Step 1: Write failing persistence and API tests**

Cover creating overrides, updating overrides, clearing with `null`, preserving explicit `false` and `0`, rejecting invalid explicit fields, and loading old JSON without `urltest`.

- [x] **Step 2: Run focused tests and confirm failure**

Run: `go test ./internal/core ./internal/api -run 'Subscription.*URLTest|SubscriptionHandlerCRUD'`

Expected: compile/test failure because method signatures and request fields are absent.

- [x] **Step 3: Implement the minimal subscription changes**

Validate overrides before persistence. Treat the submitted nullable object as a full replacement, so `null` clears all overrides. Update all existing call sites with `nil` where no override is required.

- [x] **Step 4: Run focused tests**

Run: `go test ./internal/core ./internal/api -run 'Subscription.*URLTest|SubscriptionHandlerCRUD'`

Expected: PASS.

---

### Task 4: Resolve defaults during outbound synchronization

**Files:**
- Modify: `internal/api/nodes_sync.go`
- Test: `internal/api/nodes_subscription_test.go`

**Interfaces:**
- Consumes: `SettingsManager.URLTestDefaults()` and `ResolveURLTest`.
- Produces: subscription groups whose URL, interval, tolerance, and existence match the resolved configuration.

- [x] **Step 1: Write failing synchronization tests**

Cover global defaults, partial subscription overrides, global disabled plus subscription enabled, subscription disabled, old managed group removal, and selector membership/default selection when some groups are disabled.

- [x] **Step 2: Run focused tests and confirm failure**

Run: `go test ./internal/api -run 'SyncOutbounds.*URLTest|CreatesSubscriptionGroups'`

Expected: FAIL because synchronization still writes hard-coded values.

- [x] **Step 3: Replace hard-coded values with resolved settings**

Load typed defaults from the shared bbolt database before rebuilding outbounds. Return settings decode errors. Continue collecting all subscription names for stale group cleanup, but only append enabled group tags to the proxy selector.

- [x] **Step 4: Run focused tests**

Run: `go test ./internal/api -run 'SyncOutbounds.*URLTest|CreatesSubscriptionGroups'`

Expected: PASS.

---

### Task 5: Frontend API types and mocks

**Files:**
- Modify: `ui/src/lib/api/types.ts`
- Modify: `ui/src/lib/api/endpoints.ts`
- Modify: `ui/src/lib/api/endpoints.test.ts`
- Modify: `ui/src/test/mock-api.ts`

**Interfaces:**
- Produces: `URLTestDefaults`, `URLTestOverrides`, extended `Subscription` and `SubscriptionInput`.
- Produces: `api.settings.urlTestDefaults()` and `api.settings.setURLTestDefaults(input)`.

- [x] **Step 1: Write failing endpoint tests**

Verify request paths, PUT body, subscription create/update bodies with omitted, partial, `false`, and `0` override values.

- [x] **Step 2: Run focused tests and confirm failure**

Run: `cd ui && npm test -- src/lib/api/endpoints.test.ts --run`

Expected: TypeScript/test failure because typed endpoints do not exist.

- [x] **Step 3: Implement types, endpoint methods, and mock defaults**

Use optional properties for override fields and `urltest?: URLTestOverrides | null` on input. Keep global settings fully required.

- [x] **Step 4: Run focused tests**

Run: `cd ui && npm test -- src/lib/api/endpoints.test.ts --run`

Expected: PASS.

---

### Task 6: shadcn global defaults form

**Files:**
- Modify: `ui/src/features/settings/settings-page.tsx`
- Modify: `ui/src/features/settings/settings-page.test.tsx`
- Modify: `ui/src/features/settings/settings-interactions.test.tsx`
- Modify: `ui/src/features/settings/settings-branches.test.tsx`
- Modify: `ui/src/i18n/locales/zh.ts`
- Modify: `ui/src/i18n/locales/en.ts`

**Interfaces:**
- Consumes: typed settings endpoints and existing `api.nodes.sync()`.
- Produces: a shadcn card for enabled, URL, interval, and tolerance defaults.

- [x] **Step 1: Inspect shadcn project context and component docs**

Run: `cd ui && npx shadcn@latest info && npx shadcn@latest docs card field input switch button`

Expected: confirms installed component APIs and the current Base UI preset.

- [x] **Step 2: Write failing UI tests**

Verify loading values, editing all fields, saving then synchronizing, validation messages, sync failure toast, and the explanatory distinction from TCP/HTTP/ICMP tests.

- [x] **Step 3: Run focused tests and confirm failure**

Run: `cd ui && npm test -- src/features/settings --run`

Expected: FAIL because the card is absent.

- [x] **Step 4: Implement the shadcn card**

Compose existing `Card`, `Field`, `Input`, `Switch`, and `Button`; use `flex`/`grid` with `gap-*`, semantic tokens, no custom global CSS, and no new dependencies. Validate URL, positive duration text, and tolerance `0..65535` before mutation.

- [x] **Step 5: Run focused tests**

Run: `cd ui && npm test -- src/features/settings --run`

Expected: PASS.

---

### Task 7: shadcn subscription override form

**Files:**
- Modify: `ui/src/features/subscriptions/subscriptions-page.tsx`
- Modify: `ui/src/features/subscriptions/subscriptions-page.test.tsx`
- Create: `ui/src/features/subscriptions/subscription-urltest.test.tsx`
- Modify: `ui/src/i18n/locales/zh.ts`
- Modify: `ui/src/i18n/locales/en.ts`

**Interfaces:**
- Consumes: global defaults query and extended subscription input.
- Produces: inherit/enabled/disabled policy, optional URL/interval/tolerance overrides, reset action, and subscription status badge.

- [x] **Step 1: Write failing dialog tests**

Verify create defaults to inheritance, placeholders show global values, partial overrides omit blank fields, `false` and `0` survive, edit restores custom values, reset submits `null`, and mobile dialog content remains accessible.

- [x] **Step 2: Run focused tests and confirm failure**

Run: `cd ui && npm test -- src/features/subscriptions --run`

Expected: FAIL because controls and badges are absent.

- [x] **Step 3: Implement the form and status badge**

Use existing shadcn `ToggleGroup`, `Field`, `Input`, `Button`, `Badge`, and scroll-safe Dialog composition. Derive request data directly during submit rather than mirroring derived state in effects. Fetch defaults through TanStack Query so settings and subscription views share cached data.

- [x] **Step 4: Run focused tests**

Run: `cd ui && npm test -- src/features/subscriptions --run`

Expected: PASS.

---

### Task 8: Documentation, quality gates, commit, and deployment

**Files:**
- Modify: `README.md`
- Modify: `docs/boxui/plans/2026-07-15-urltest-inheritance.md`

- [x] **Step 1: Update concise user documentation**

Mention global URLTest defaults and per-subscription inheritance under the existing node/subscription and application settings capability bullets.

- [x] **Step 2: Run formatting and focused regression tests**

Run:

```bash
goimports-reviser -rm-unused -set-alias -project-name github.com/xuthus5/boxui -recursive ./internal
goimports-reviser -rm-unused -set-alias -project-name github.com/xuthus5/boxui -recursive ./cmd
go test ./internal/core ./internal/api
cd ui && npm run typecheck && npm run lint && npm run coverage
```

Expected: all commands exit 0 and UI business coverage remains at least 90%.

- [x] **Step 3: Run full project gates**

Run:

```bash
make check-go
make check-ui
make build
make check-embedded-ui
```

Expected: all commands exit 0; temporary embedded UI copy is cleaned by the build.

- [x] **Step 4: Review the final diff and commit**

Run: `git diff --check && git status --short`

Commit: `feat(urltest): add inherited defaults`

- [x] **Step 5: Deploy the committed binary locally**

Safely install `bin/boxui` as `/usr/local/bin/boxui` with owner `root:boxui` and mode `0750`, restart `boxui.service`, verify `active (running)`, verify the process executable, and inspect current-boot service logs for startup failures.
