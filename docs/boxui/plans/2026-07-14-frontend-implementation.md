# BoxUI Frontend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a production-ready React 19 and TypeScript administration panel that covers every approved BoxUI capability through the existing Go API without backend changes.

**Architecture:** A Vite SPA uses React Router for protected lazy routes, TanStack Query for HTTP state, a typed API client for direct and enveloped responses, and authenticated Fetch Streams for SSE. Business modules own focused pages, hooks, schemas, and tests while shared UI is composed exclusively from shadcn/ui components.

**Tech Stack:** React 19, TypeScript, Vite, shadcn/ui, Tailwind CSS, React Router, TanStack Query, React Hook Form, Zod, i18next, CodeMirror, Recharts, Vitest, Testing Library, MSW, Playwright.

## Global Constraints

- Do not modify Go backend behavior or add backend endpoints.
- Use npm, matching the existing README and Makefile commands.
- Do not add global visual styling beyond the shadcn-generated Tailwind theme and base layer.
- Use shadcn semantic tokens and built-in variants; layout-only utility classes are allowed.
- Preserve unknown sing-box JSON fields during structured edits.
- Keep functions at or below 50 lines, files at or below 300 lines, nesting at or below 3, and positional parameters at or below 3.
- Keep lines, functions, statements, and branches coverage at or above 90%.
- Create directories with mode `0700` and files with mode `0600`.
- Do not commit, rewrite Git history, or contact remotes without explicit user authorization.

---

### Task 1: Scaffold the React, shadcn, and test toolchain

**Files:**
- Create: `ui/package.json`
- Create: `ui/package-lock.json`
- Create: `ui/index.html`
- Create: `ui/components.json`
- Create: `ui/tsconfig.json`
- Create: `ui/tsconfig.app.json`
- Create: `ui/tsconfig.node.json`
- Create: `ui/vite.config.ts`
- Create: `ui/vitest.config.ts`
- Create: `ui/eslint.config.js`
- Create: `ui/playwright.config.ts`
- Create: `ui/src/main.tsx`
- Create: `ui/src/app.tsx`
- Create: `ui/src/index.css`
- Create: `ui/src/test/setup.ts`
- Create: `ui/src/test/render.tsx`
- Create: `ui/src/test/smoke.test.tsx`
- Create: shadcn component sources under `ui/src/components/ui/`

**Interfaces:**
- Produces: `renderApp(ui: ReactElement, route?: string): RenderResult`, npm scripts `dev`, `build`, `typecheck`, `lint`, `test`, `coverage`, `check`, `e2e`, and the `@/` alias.

- [ ] **Step 1: Initialize Vite and install exact capability groups**

Run from the repository root:

```bash
npm create vite@latest ui -- --template react-ts
cd ui
npm install react-router-dom @tanstack/react-query react-hook-form @hookform/resolvers zod i18next react-i18next recharts @uiw/react-codemirror @codemirror/lang-json lucide-react sonner
npm install -D vitest @vitest/coverage-v8 jsdom @testing-library/react @testing-library/jest-dom @testing-library/user-event msw @playwright/test eslint @eslint/js typescript-eslint eslint-plugin-react-hooks eslint-plugin-react-refresh
npx shadcn@latest init
```

Expected: `ui/components.json` exists and reports a Vite project with the configured shadcn base.

- [ ] **Step 2: Add only the required shadcn components through the CLI**

Run:

```bash
npx shadcn@latest add alert alert-dialog badge button card chart checkbox dialog dropdown-menu empty field input input-group label scroll-area select separator sheet sidebar skeleton sonner switch table tabs textarea toggle-group tooltip
```

Expected: every component is created under `ui/src/components/ui/`; no hand-written substitute duplicates these components.

- [ ] **Step 3: Write the failing application smoke test**

```tsx
import { screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"
import { renderApp } from "@/test/render"
import { App } from "@/app"

describe("App", () => {
  it("renders the login route without an authenticated session", () => {
    renderApp(<App />, "/")
    expect(screen.getByRole("heading", { name: /boxui/i })).toBeInTheDocument()
  })
})
```

- [ ] **Step 4: Configure the providers and minimal login placeholder to pass**

`renderApp` wraps children in `MemoryRouter`, `QueryClientProvider`, and `I18nextProvider`. `App` renders a semantic login heading only until Task 3 replaces it with routing.

- [ ] **Step 5: Run the scaffold gate**

Run: `cd ui && npm run typecheck && npm run lint && npm run test -- --run`

Expected: exit code `0`, one smoke test passing, and no lint errors.

### Task 2: Implement typed API, session, and authenticated SSE foundations

**Files:**
- Create: `ui/src/lib/api/types.ts`
- Create: `ui/src/lib/api/client.ts`
- Create: `ui/src/lib/api/endpoints.ts`
- Create: `ui/src/lib/api/sse.ts`
- Create: `ui/src/lib/session.ts`
- Create: `ui/src/lib/storage.ts`
- Create: `ui/src/lib/api/client.test.ts`
- Create: `ui/src/lib/api/sse.test.ts`
- Create: `ui/src/lib/session.test.ts`

**Interfaces:**
- Produces: `apiRequest<T>(path: string, init?: RequestInit): Promise<T>`.
- Produces: `openSSE<T>(options: SSEOptions<T>): () => void` where `SSEOptions<T>` contains `path`, `token`, `signal`, `onEvent`, and `onError`.
- Produces: `sessionStore.get()`, `sessionStore.set(session)`, `sessionStore.clear()`, and `sessionStore.isValid()`.
- Produces: endpoint functions for every route defined in `internal/api/router.go`.

- [ ] **Step 1: Write response normalization tests**

Cover direct JSON, `{status,data,error,meta}`, empty `204`, invalid JSON, `401`, and `429`. Assert that errors become `ApiError` instances with `status`, `code`, and `message`.

- [ ] **Step 2: Run the client tests and verify failure**

Run: `cd ui && npm run test -- src/lib/api/client.test.ts --run`

Expected: FAIL because `apiRequest` and `ApiError` do not exist.

- [ ] **Step 3: Implement the API contract**

Define exact shared types matching Go JSON tags: `AuthResponse`, `ServiceStatus`, `TrafficEvent`, `TrafficHistoryPoint`, `LogEvent`, `Connection`, `ConnectionEvent`, `Subscription`, `Outbound`, `ImportResult`, `TestResult`, `APIEnvelope<T>`, and `ApiErrorBody`.

`apiRequest` must add `Accept: application/json`, add `Content-Type` only when a body exists, attach the current bearer token, unwrap successful envelopes, and call a registered unauthorized handler after constructing a `401` error.

- [ ] **Step 4: Write SSE framing tests**

Feed encoded chunks containing split `data:` lines, multiple events, comments, malformed JSON, abort, and a retryable network failure. Assert ordered parsed events and exactly one error callback per failed connection.

- [ ] **Step 5: Implement authenticated Fetch Stream SSE**

Use `fetch(path, {headers: {Authorization: `Bearer ${token}`}, signal})`, a `TextDecoder`, and an internal line buffer. Return a cleanup function that aborts the active controller and prevents reconnection.

- [ ] **Step 6: Implement and test versioned storage**

Store sessions under `boxui.session.v1` in `sessionStorage`; store theme and language under `boxui.preferences.v1` in `localStorage`. Catch unavailable or malformed storage and fall back to defaults without throwing.

- [ ] **Step 7: Run foundation tests**

Run: `cd ui && npm run test -- src/lib --run`

Expected: all API, SSE, session, and storage tests pass.

### Task 3: Build authentication, localization, theme, routing, and the shadcn shell

**Files:**
- Create: `ui/src/app/providers.tsx`
- Create: `ui/src/app/router.tsx`
- Create: `ui/src/app/navigation.ts`
- Create: `ui/src/app/app-shell.tsx`
- Create: `ui/src/app/protected-route.tsx`
- Create: `ui/src/features/auth/login-page.tsx`
- Create: `ui/src/features/auth/auth-hooks.ts`
- Create: `ui/src/features/preferences/preferences-provider.tsx`
- Create: `ui/src/i18n/index.ts`
- Create: `ui/src/i18n/locales/zh.ts`
- Create: `ui/src/i18n/locales/en.ts`
- Create: `ui/src/app/router.test.tsx`
- Create: `ui/src/features/auth/login-page.test.tsx`
- Modify: `ui/src/app.tsx`

**Interfaces:**
- Consumes: `api.auth.login`, `api.auth.logout`, and `sessionStore` from Task 2.
- Produces: protected lazy routes, `usePreferences()`, and the final sidebar navigation model.

- [ ] **Step 1: Write failing login and route protection tests**

Assert validation messages for empty credentials, successful token persistence, error display for `401` and `429`, redirect from protected routes, logout cleanup, and redirect after expiration.

- [ ] **Step 2: Implement the login flow**

Use `Card`, `FieldGroup`, `Field`, `Input`, `Button`, `Alert`, and `Spinner`. Validate username and password with Zod, disable submission while pending, and never log credentials.

- [ ] **Step 3: Implement theme and language providers**

Support `light`, `dark`, and `system`. Apply only the shadcn `dark` class to the document root. Register complete Chinese and English translation objects with identical keys.

- [ ] **Step 4: Implement the responsive application shell**

Use shadcn `SidebarProvider`, `Sidebar`, `SidebarContent`, `SidebarGroup`, `SidebarMenu`, `SidebarInset`, and mobile trigger. Navigation contains the approved hierarchy; kernel and application logs do not appear as navigation items.

- [ ] **Step 5: Run authentication and shell tests**

Run: `cd ui && npm run test -- src/features/auth src/app --run`

Expected: all route, navigation, login, logout, theme, and language tests pass.

### Task 4: Implement shared configuration editing primitives

**Files:**
- Create: `ui/src/features/config/config-types.ts`
- Create: `ui/src/features/config/config-api.ts`
- Create: `ui/src/features/config/config-hooks.ts`
- Create: `ui/src/features/config/config-update.ts`
- Create: `ui/src/features/config/json-editor.tsx`
- Create: `ui/src/features/config/json-dialog.tsx`
- Create: `ui/src/features/config/config-update.test.ts`
- Create: `ui/src/features/config/json-editor.test.tsx`

**Interfaces:**
- Produces: `SingBoxConfig = Record<string, JsonValue>` and JSON-safe recursive types.
- Produces: `replaceConfigSection(config, key, value): SingBoxConfig` without mutating input.
- Produces: `replaceConfigArrayItem(config, key, index, item): SingBoxConfig` preserving unknown fields.
- Produces: `JsonEditor` with `value`, `onChange`, `ariaLabel`, and `readOnly` props.
- Produces: `useConfigQuery()` and `useSaveConfigMutation()`.

- [ ] **Step 1: Write failing immutable update tests**

Assert adding, replacing, deleting, and reordering array entries; preserve unrelated top-level and nested fields; reject non-object JSON for item editing.

- [ ] **Step 2: Implement JSON-safe update helpers**

Use shallow copies only along changed paths. Do not stringify the entire configuration to clone it because that would obscure unsupported values and waste work.

- [ ] **Step 3: Write failing editor tests**

Assert valid JSON formatting, invalid JSON diagnostics, disabled save state, reset to server value, and retention of unsaved text after an API error.

- [ ] **Step 4: Implement the CodeMirror wrapper and save mutation**

Use the JSON language extension and shadcn `Alert` for validation. Interpret response status `rolled_back` as a typed non-success result shown to the caller.

- [ ] **Step 5: Run configuration primitive tests**

Run: `cd ui && npm run test -- src/features/config --run`

Expected: all immutable update and editor tests pass.

### Task 5: Implement the dashboard

**Files:**
- Create: `ui/src/features/dashboard/dashboard-page.tsx`
- Create: `ui/src/features/dashboard/dashboard-api.ts`
- Create: `ui/src/features/dashboard/service-card.tsx`
- Create: `ui/src/features/dashboard/traffic-chart.tsx`
- Create: `ui/src/features/dashboard/runtime-actions.tsx`
- Create: `ui/src/features/dashboard/recent-logs.tsx`
- Create: `ui/src/features/dashboard/dashboard-page.test.tsx`
- Create: `ui/src/features/dashboard/traffic-chart.test.tsx`

**Interfaces:**
- Consumes: service, traffic history, traffic SSE, log SSE, memory, version, GC, DNS flush, and FakeIP flush endpoints.
- Produces: lazy route component `DashboardPage`.

- [ ] **Step 1: Write failing dashboard state tests**

Cover running and stopped service, action pending state, mutation error, traffic history plus live append, bounded chart points, runtime maintenance actions, and recent log rendering.

- [ ] **Step 2: Implement dashboard queries in parallel**

Start independent service, history, memory, and version queries together through TanStack Query. Append SSE samples to a fixed-size buffer without sorting on every event.

- [ ] **Step 3: Compose the page from shadcn components**

Use full `Card` composition, shadcn `ChartContainer`, `Badge`, `Button`, `Alert`, and `Skeleton`. Service actions must display confirmation for stop and restart.

- [ ] **Step 4: Run dashboard tests**

Run: `cd ui && npm run test -- src/features/dashboard --run`

Expected: dashboard tests pass with no unhandled requests.

### Task 6: Implement proxy configuration and traffic policy modules

**Files:**
- Create: `ui/src/features/proxy/inbounds-page.tsx`
- Create: `ui/src/features/proxy/outbounds-page.tsx`
- Create: `ui/src/features/proxy/proxy-list.tsx`
- Create: `ui/src/features/proxy/proxy-form.tsx`
- Create: `ui/src/features/proxy/proxy-schema.ts`
- Create: `ui/src/features/proxy/proxy-pages.test.tsx`
- Create: `ui/src/features/policy/route-page.tsx`
- Create: `ui/src/features/policy/dns-page.tsx`
- Create: `ui/src/features/policy/policy-list-editor.tsx`
- Create: `ui/src/features/policy/policy-pages.test.tsx`

**Interfaces:**
- Consumes: Task 4 config hooks and default installer endpoints.
- Produces: `InboundsPage`, `OutboundsPage`, `RoutePage`, and `DNSPage` lazy route components.

- [ ] **Step 1: Write failing proxy behavior tests**

Cover empty state, add, edit, delete confirmation, reorder, common field validation, advanced JSON preservation, default outbound installation, rollback response, and server validation error.

- [ ] **Step 2: Implement common proxy forms**

Expose common `type`, `tag`, `listen`, `listen_port`, `server`, and `server_port` fields only when applicable. Merge form values into the original raw object so protocol-specific fields survive.

- [ ] **Step 3: Write failing route and DNS tests**

Cover rule and server list operations, final outbound, default route, rule-set and DNS installers, unknown field preservation, and save failure.

- [ ] **Step 4: Implement policy pages**

Use `Table` for ordered entries, `Sheet` for structured editing, `JsonDialog` for advanced data, and `AlertDialog` for destructive actions.

- [ ] **Step 5: Run proxy and policy tests**

Run: `cd ui && npm run test -- src/features/proxy src/features/policy --run`

Expected: all proxy and policy tests pass.

### Task 7: Implement nodes and subscriptions

**Files:**
- Create: `ui/src/features/nodes/nodes-api.ts`
- Create: `ui/src/features/nodes/nodes-page.tsx`
- Create: `ui/src/features/nodes/import-node-dialog.tsx`
- Create: `ui/src/features/nodes/node-editor.tsx`
- Create: `ui/src/features/nodes/node-tests.tsx`
- Create: `ui/src/features/nodes/nodes-page.test.tsx`
- Create: `ui/src/features/subscriptions/subscriptions-api.ts`
- Create: `ui/src/features/subscriptions/subscriptions-page.tsx`
- Create: `ui/src/features/subscriptions/subscription-dialog.tsx`
- Create: `ui/src/features/subscriptions/subscriptions-page.test.tsx`

**Interfaces:**
- Produces: typed CRUD, import, test, batch test, sync, selector, URLTest, and subscription refresh hooks.
- Produces: `NodesPage` and `SubscriptionsPage` lazy routes.

- [ ] **Step 1: Write failing node workflow tests**

Cover link preview, save, edit, delete, tcp/http/icmp tests, batch concurrency payload, persisted result display, sync-to-config, selector change, URLTest result, and row-level error isolation.

- [ ] **Step 2: Implement node workflows**

Use Tables for nodes and results, Dialog for import, Sheet for editing, ToggleGroup for test type, and Sonner for mutation feedback. Never expose credentials outside the editing surface.

- [ ] **Step 3: Write failing subscription workflow tests**

Cover create, update, delete, refresh one, refresh all, last-updated formatting, node count, and backend error text.

- [ ] **Step 4: Implement subscription workflows**

Use validated name, URL, and interval fields. Invalidate both subscription and node queries after successful refresh.

- [ ] **Step 5: Run node and subscription tests**

Run: `cd ui && npm run test -- src/features/nodes src/features/subscriptions --run`

Expected: all node and subscription tests pass.

### Task 8: Implement active connections and the tabbed logs page

**Files:**
- Create: `ui/src/features/observability/connections-page.tsx`
- Create: `ui/src/features/observability/connections-table.tsx`
- Create: `ui/src/features/observability/logs-page.tsx`
- Create: `ui/src/features/observability/log-panel.tsx`
- Create: `ui/src/features/observability/use-stream-buffer.ts`
- Create: `ui/src/features/observability/use-stream-buffer.test.ts`
- Create: `ui/src/features/observability/connections-page.test.tsx`
- Create: `ui/src/features/observability/logs-page.test.tsx`

**Interfaces:**
- Consumes: connection, kernel log, and application log SSE endpoints plus connection delete endpoints.
- Produces: `ConnectionsPage` and `LogsPage` lazy routes.

- [ ] **Step 1: Write failing bounded stream buffer tests**

Assert append, maximum length, clear, pause, resume, filter, independent buffers, and cleanup after unmount.

- [ ] **Step 2: Implement the reusable stream buffer hook**

Use functional state updates and stable callbacks. Keep transient auto-follow state in a ref to avoid rerendering for scroll events.

- [ ] **Step 3: Write failing connection tests**

Cover live list replacement, byte formatting, empty state, close one, close all confirmation, stream error, and reconnect status.

- [ ] **Step 4: Write failing log page tests**

Assert the sidebar has only one log route, the page has kernel and application Tabs, each Tab retains its own filter and buffer, and pause or clear affects only the active source.

- [ ] **Step 5: Implement observability pages**

Use `TabsList` and `TabsTrigger` only inside the log page. Use `ScrollArea`, `Input`, `Select`, `Switch`, `Badge`, `Table`, `Empty`, and `Alert` without custom log bubble markup.

- [ ] **Step 6: Run observability tests**

Run: `cd ui && npm run test -- src/features/observability --run`

Expected: all connection and log tests pass.

### Task 9: Implement advanced configuration and application settings

**Files:**
- Create: `ui/src/features/advanced/section-config-page.tsx`
- Create: `ui/src/features/advanced/endpoints-page.tsx`
- Create: `ui/src/features/advanced/experimental-page.tsx`
- Create: `ui/src/features/advanced/raw-config-page.tsx`
- Create: `ui/src/features/advanced/advanced-pages.test.tsx`
- Create: `ui/src/features/settings/settings-api.ts`
- Create: `ui/src/features/settings/settings-page.tsx`
- Create: `ui/src/features/settings/password-card.tsx`
- Create: `ui/src/features/settings/jwt-card.tsx`
- Create: `ui/src/features/settings/runtime-settings-card.tsx`
- Create: `ui/src/features/settings/settings-page.test.tsx`

**Interfaces:**
- Consumes: Task 4 JSON editor and all `/api/settings` endpoints.
- Produces: `EndpointsPage`, `ExperimentalPage`, `RawConfigPage`, and `SettingsPage` lazy routes.

- [ ] **Step 1: Write failing advanced configuration tests**

Cover missing sections, valid edit, invalid JSON, reset, full-config save confirmation, server validation error, rollback, and unknown section preservation.

- [ ] **Step 2: Implement advanced pages**

Reuse `SectionConfigPage` for endpoints and experimental. The raw page fetches `/api/config/raw`, formats with two-space indentation, and submits the exact parsed JSON value.

- [ ] **Step 3: Write failing settings tests**

Cover theme, language, read-only username explanation, default-password warning, current-password failure, weak-password failure, successful password logout, JWT status, JWT rotation logout, test URL reset, and kernel autostart.

- [ ] **Step 4: Implement settings cards**

Use separate full Card compositions. Passwords and secrets use password inputs, are never persisted, and are cleared after any mutation completes. JWT rotation uses an AlertDialog explaining that all sessions are invalidated.

- [ ] **Step 5: Run advanced and settings tests**

Run: `cd ui && npm run test -- src/features/advanced src/features/settings --run`

Expected: all advanced configuration and settings tests pass.

### Task 10: Complete integration, coverage, documentation, and embedded build verification

**Files:**
- Create: `ui/src/test/server.ts`
- Create: `ui/src/test/handlers.ts`
- Create: `ui/e2e/app.spec.ts`
- Modify: `ui/vitest.config.ts`
- Modify: `ui/playwright.config.ts`
- Modify: `ui/package.json`
- Modify: `README.md`

**Interfaces:**
- Consumes: all earlier routes and API modules.
- Produces: deterministic MSW defaults, isolated Playwright smoke coverage, final `npm run check`, and documented frontend commands.

- [ ] **Step 1: Add complete default MSW handlers**

Every HTTP endpoint in `internal/api/router.go` must have a default authenticated handler. Tests override only behavior relevant to the current case. Fail tests on unhandled requests.

- [ ] **Step 2: Add browser smoke tests**

Test login, sidebar navigation, dashboard rendering, log Tab switching, and raw configuration save. Start only isolated test processes and ensure Playwright teardown terminates them.

- [ ] **Step 3: Enforce coverage thresholds**

Configure V8 coverage with `lines: 90`, `functions: 90`, `statements: 90`, and `branches: 90`. Exclude generated shadcn component sources, type-only files, Vite bootstrap, and test utilities; do not exclude business modules.

- [ ] **Step 4: Update the concise README**

Keep the existing quick-start commands, state that the UI now lives in `ui/`, and document `npm run check` plus `npm run e2e` without duplicating the design document.

- [ ] **Step 5: Run frontend verification**

Run:

```bash
cd ui
npm run check
npm run build
npm run e2e
```

Expected: typecheck, lint, all tests, all four coverage thresholds, production build, and browser smoke tests exit `0`.

- [ ] **Step 6: Run repository verification**

Run from the repository root:

```bash
go test ./...
golangci-lint run ./...
goimports-reviser -rm-unused -set-alias -local github.com/xuthus5/boxui -project-path ./internal ./cmd
make check-embedded-ui
```

Expected: every command exits `0`; `goimports-reviser` produces no backend changes because the frontend task does not modify Go files.

- [ ] **Step 7: Clean temporary artifacts and audit the result**

Remove only generated test artifacts such as `ui/coverage/`, `ui/playwright-report/`, `ui/test-results/`, `ui/dist/`, and temporary embedded copies. Preserve `package-lock.json`. Verify file modes, confirm no file exceeds 300 lines, confirm no function exceeds 50 lines, and inspect `git diff --stat` plus `git status --short`.
