# Route and DNS Form Editor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the route and DNS JSON-only pages with shadcn card and form editors while preserving complete JSON editing, unknown fields, existing APIs, default installers, rollback behavior, and local deployment.

**Architecture:** Introduce policy-specific immutable JSON helpers and field rendering primitives, then build independent route and DNS metadata models and visual editors on top. Keep the shared page shell responsible only for loading, whole-section JSON synchronization, saving, default installation, and visual/advanced tab composition.

**Tech Stack:** React 19, TypeScript 6, Vite, shadcn/Base UI, Tailwind CSS v4, TanStack Query, Vitest, Testing Library, Playwright.

**Source Spec:** `docs/boxd/specs/2026-07-15-policy-form-editor-design.md`

## Global Constraints

- Do not modify backend code, API contracts, default installer behavior, or sing-box startup logic.
- Target the configuration shapes supported by sing-box `v1.13.14`.
- Use only existing shadcn/Base UI components and semantic Tailwind layout classes; do not add global custom styles.
- Preserve unknown top-level and nested JSON fields during all structured edits.
- Do not migrate legacy DNS server or top-level FakeIP configuration into modern shapes.
- Route and DNS rules use up/down controls only; do not add drag-and-drop or dependencies.
- Keep files at or below 300 lines, functions at or below 50 lines, nesting at or below 3, and cyclomatic complexity at or below 10.
- Maintain statements, branches, functions, and lines coverage at or above 90%.
- Finish with code review, Git commits using the repository convention, production build, binary replacement, and `boxd.service` verification.

---

### Task 1: Add policy JSON and field primitives

**Files:**

- Create: `ui/src/features/policy/policy-form-model.ts`
- Create: `ui/src/features/policy/policy-form-fields.tsx`
- Create: `ui/src/features/policy/policy-form.test.tsx`

**Interfaces:**

```ts
export type JsonObject = Record<string, JsonValue>
export type PolicyFieldKind = "text" | "textarea" | "number" | "boolean" | "list" | "number-list" | "select" | "json-object" | "json-array"
export interface PolicyFieldSpec {
  path: string
  label: string
  kind?: PolicyFieldKind
  options?: readonly string[]
  required?: boolean
}
export type PolicyFieldTransform = (object: JsonObject, field: PolicyFieldSpec, raw: string) => JsonObject | null | undefined

export function isJsonObject(value: JsonValue | undefined): value is JsonObject
export function getPolicyPath(object: JsonObject, path: string): JsonValue | undefined
export function setPolicyPath(object: JsonObject, path: string, value: JsonValue | undefined): JsonObject
export function moveItem<T>(items: readonly T[], index: number, direction: -1 | 1): T[]
export function cloneJsonObject(object: JsonObject): JsonObject
```

`PolicyFormFields` consumes field metadata, a complete object, namespace, optional revision, change callback, validity callback, and optional field transform. It renders shadcn fields without protocol-specific rules.

- [ ] **Step 1: Write failing immutable helper tests**

```ts
expect(getPolicyPath({ action: { server: "dns-remote" } }, "action.server")).toBe("dns-remote")
expect(setPolicyPath({}, "action.server", "dns-remote")).toEqual({ action: { server: "dns-remote" } })
expect(setPolicyPath({ action: { server: "dns-remote" } }, "action.server", undefined)).toEqual({})
expect(moveItem(["a", "b", "c"], 1, -1)).toEqual(["b", "a", "c"])
expect(moveItem(["a", "b"], 0, -1)).toEqual(["a", "b"])
```

- [ ] **Step 2: Run tests and verify RED**

```bash
npm --prefix ui test -- --run src/features/policy/policy-form.test.tsx
```

Expected: FAIL because the new modules do not exist.

- [ ] **Step 3: Implement immutable helpers**

Use recursive copy-on-write path updates, prune empty parents, and never mutate source arrays:

```ts
export function moveItem<T>(items: readonly T[], index: number, direction: -1 | 1) {
  const target = index + direction
  if (index < 0 || index >= items.length || target < 0 || target >= items.length) return [...items]
  const next = [...items]
  ;[next[index], next[target]] = [next[target], next[index]]
  return next
}
```

- [ ] **Step 4: Write failing field conversion tests**

Cover text, number, boolean, list, number-list, stable Base UI Select items, JSON object/array validation, invalid cleanup, and revision reset after Advanced JSON changes another field.

```tsx
<PolicyFormFields
  fields={[{ path: "headers", label: "headers", kind: "json-object" }]}
  object={{ headers: { X: "old" }, tag: "changed" }}
  namespace="policy.dns"
  revision={1}
  onChange={setObject}
  onFieldValidityChange={onValidity}
/>
```

- [ ] **Step 5: Implement `PolicyFormFields`**

Compose `FieldGroup`, `Field`, `FieldLabel`, `FieldDescription`, `Input`, `Textarea`, `Switch`, `Select`, `SelectContent`, `SelectGroup`, `SelectItem`, `SelectTrigger`, and `SelectValue`. Parse list fields by newline/comma, reject non-finite numbers, and synchronize structured drafts with `${revision}:${serializedValue}`.

- [ ] **Step 6: Verify and commit**

```bash
npm --prefix ui test -- --run src/features/policy/policy-form.test.tsx
npm --prefix ui run typecheck
npm --prefix ui run lint
git add ui/src/features/policy/policy-form-model.ts ui/src/features/policy/policy-form-fields.tsx ui/src/features/policy/policy-form.test.tsx
git commit -m "refactor(policy): add form primitives"
```

Expected: all checks pass and the commit succeeds.

---

### Task 2: Replace the JSON-only policy shell

**Files:**

- Modify: `ui/src/features/policy/policy-page.tsx`
- Modify: `ui/src/features/policy/policy-state.test.tsx`
- Create: `ui/src/features/policy/policy-page.test.tsx`

**Interfaces:**

```ts
export interface PolicyVisualEditorProps {
  object: JsonObject
  revision: number
  onChange: (object: JsonObject) => void
  onFieldValidityChange: (path: string, valid: boolean) => void
}

interface PolicyPageProps {
  section: "route" | "dns"
  title: string
  installLabel: string
  install: () => Promise<APIEnvelope<JsonValue>>
  renderVisual: (props: PolicyVisualEditorProps) => React.ReactNode
}
```

- [ ] **Step 1: Write failing shell tests**

Require the visual tab by default, Advanced JSON in the second tab, invalid root JSON disabling Save, revision updates reaching the visual editor, structured invalid fields disabling Save, section-only replacement in the PUT body, and unchanged install success/failure/rollback behavior.

- [ ] **Step 2: Run shell tests and verify RED**

```bash
npm --prefix ui test -- --run src/features/policy/policy-page.test.tsx src/features/policy/policy-state.test.tsx
```

Expected: FAIL because `PolicyPage` has no Tabs or `renderVisual` contract.

- [ ] **Step 3: Implement one-string source of truth**

```ts
const [value, setValue] = useState(() => JSON.stringify(initialSection, null, 2))
const [revision, setRevision] = useState(0)
const object = parsePolicyObject(value)
const updateObject = (next: JsonObject) => setValue(JSON.stringify(next, null, 2))
const updateJSON = (next: string) => {
  setValue(next)
  setRevision((current) => current + 1)
}
```

Render full Card composition and `TabsList` containing `可视化配置` and `高级 JSON`. Keep current Skeleton, Alert, toast, refetch, rollback, and install sequencing behavior.

- [ ] **Step 4: Verify and commit**

```bash
npm --prefix ui test -- --run src/features/policy/policy-page.test.tsx src/features/policy/policy-state.test.tsx
npm --prefix ui run typecheck
npm --prefix ui run lint
git add ui/src/features/policy/policy-page.tsx ui/src/features/policy/policy-page.test.tsx ui/src/features/policy/policy-state.test.tsx
git commit -m "refactor(policy): add visual editor shell"
```

Expected: all checks pass.

### Task 3: Model route globals, rules, actions, and rule sets

**Files:**

- Create: `ui/src/features/policy/route-form-model.ts`
- Create: `ui/src/features/policy/route-form-model.test.ts`

**Interfaces:**

```ts
export const routeGlobalFields: readonly PolicyFieldSpec[]
export const routeMatchFields: readonly PolicyFieldSpec[]
export const routeActions: readonly string[]
export const routeActionFields: Record<string, readonly PolicyFieldSpec[]>
export const ruleSetTypes: readonly ["inline", "local", "remote"]
export function routeRules(object: JsonObject): JsonObject[]
export function routeRuleSets(object: JsonObject): JsonObject[]
export function setRouteRules(object: JsonObject, rules: readonly JsonObject[]): JsonObject
export function setRouteRuleSets(object: JsonObject, ruleSets: readonly JsonObject[]): JsonObject
export function changeRouteRuleType(rule: JsonObject, type: string): JsonObject
export function changeRouteAction(rule: JsonObject, action: string): JsonObject
export function changeRuleSetType(ruleSet: JsonObject, type: string): JsonObject
export function summarizeRouteRule(rule: JsonObject): { matches: string[]; action: string }
export function summarizeRuleSet(ruleSet: JsonObject): { type: string; detail: string }
```

- [ ] **Step 1: Write failing metadata tests**

Assert global metadata includes `final`, `find_process`, `auto_detect_interface`, `override_android_vpn`, `default_interface`, `default_mark`, nested `default_domain_resolver`, `default_network_strategy`, `default_network_type`, `default_fallback_network_type`, and `default_fallback_delay`.

Assert match metadata includes:

```text
type, inbound, ip_version, network, auth_user, protocol, client,
domain, domain_suffix, domain_keyword, domain_regex,
source_ip_cidr, source_ip_is_private, ip_cidr, ip_is_private,
source_port, source_port_range, port, port_range,
process_name, process_path, process_path_regex, package_name,
user, user_id, rule_set, rule_set_ip_cidr_match_source,
clash_mode, network_type, network_is_expensive,
network_is_constrained, wifi_ssid, wifi_bssid, invert
```

- [ ] **Step 2: Write failing transition tests**

```ts
expect(changeRouteRuleType({ type: "logical", mode: "and", rules: [], custom: "keep" }, "default"))
  .toEqual({ custom: "keep" })
expect(changeRouteAction({ action: "reject", method: "drop", outbound: "old", custom: "keep" }, "route"))
  .toEqual({ action: "route", custom: "keep" })
expect(changeRuleSetType({ type: "remote", tag: "geo", url: "https://example/r.srs", update_interval: "1d", custom: "keep" }, "local"))
  .toEqual({ type: "local", tag: "geo", custom: "keep" })
```

Require same-type identity, unknown-field retention, and target-driven removal of known incompatible paths.

- [ ] **Step 3: Write failing summary and array tests**

```ts
expect(summarizeRouteRule({ domain_suffix: ["example.com"], network: "tcp", outbound: "proxy" }))
  .toEqual({ matches: ["example.com", "tcp"], action: "proxy" })
expect(summarizeRuleSet({ type: "remote", tag: "geoip-cn", format: "binary", url: "https://example/geoip-cn.srs" }))
  .toEqual({ type: "remote · binary", detail: "https://example/geoip-cn.srs" })
```

- [ ] **Step 4: Run model tests and verify RED**

```bash
npm --prefix ui test -- --run src/features/policy/route-form-model.test.ts
```

Expected: FAIL because the route model does not exist.

- [ ] **Step 5: Implement explicit route metadata**

```ts
export const routeActions = ["route", "route-options", "direct", "bypass", "reject", "hijack-dns", "sniff", "resolve"] as const
```

Treat omitted action plus `outbound` as the legacy route form and normalize only after explicit action selection. Preserve unknown action and rule-set types as current Select values.

- [ ] **Step 6: Verify route model**

```bash
npm --prefix ui test -- --run src/features/policy/route-form-model.test.ts
```

Expected: PASS.

---

### Task 4: Build route cards and dialogs

**Files:**

- Create: `ui/src/features/policy/route-global-card.tsx`
- Create: `ui/src/features/policy/route-rule-card.tsx`
- Create: `ui/src/features/policy/route-rule-dialog.tsx`
- Create: `ui/src/features/policy/route-rule-set-card.tsx`
- Create: `ui/src/features/policy/route-rule-set-dialog.tsx`
- Create: `ui/src/features/policy/route-visual-editor.tsx`
- Create: `ui/src/features/policy/route-editor.test.tsx`
- Modify: `ui/src/features/policy/route-page.tsx`

**Interfaces:**

```ts
export function RouteVisualEditor(props: PolicyVisualEditorProps): React.ReactNode
interface RouteRuleDialogProps {
  open: boolean
  item: JsonObject
  title: string
  onOpenChange: (open: boolean) => void
  onSave: (item: JsonObject) => void
}
```

- [ ] **Step 1: Write failing global card tests**

Render `RoutePage` with all global fields plus `custom`. Change final, switches, resolver, network types, and delay; save and assert the PUT body preserves `custom` and the rest of the full config.

- [ ] **Step 2: Write failing rule workflow tests**

Cover Empty/Add, index and summary badges, default-rule creation, all approved match tabs, each supported action, deep-copy insertion, AlertDialog delete, adjacent movement, boundary disabled states, logical mode/invert with child rules in Advanced JSON, and invalid JSON blocking Dialog Save across tabs.

- [ ] **Step 3: Write failing rule-set tests**

Cover remote/local/inline/unknown types. Remote requires Tag+URL, local Tag+path, inline Tag. Changing type removes incompatible known paths, preserves unknown keys, and rule-set cards never show move controls.

- [ ] **Step 4: Run route UI tests and verify RED**

```bash
npm --prefix ui test -- --run src/features/policy/route-editor.test.tsx
```

Expected: FAIL because the route UI does not exist.

- [ ] **Step 5: Implement route cards**

Use full Card composition, `Badge` for at most three match summaries plus action, visible Edit, and small Copy/Up/Down/Delete actions. On mobile place secondary actions in `DropdownMenu`. Use `Empty` for empty sections and `AlertDialog` for deletion.

- [ ] **Step 6: Implement the route rule Dialog**

Use `sm:max-w-5xl`, internal scrolling, required `DialogTitle`, and Tabs:

```text
基础与网络 | 域名与地址 | 端口与进程 | 规则集与网络环境 | 执行动作 | 高级 JSON
```

Keep structured JSON panels mounted. Disable confirmation for non-object roots, invalid structured fields, and missing action-specific required values.

- [ ] **Step 7: Implement rule-set cards and Dialog**

Use type-driven source fields. Inline rule sets provide structured headless-rule cards and dialogs. Route and DNS logical rules manage direct children as cards and dialogs; a child logical rule keeps JSON editing for its own nested rules. Unknown fields retain Advanced JSON fallback. Cards show Tag, type, format, path/URL, or inline-rule count.

- [ ] **Step 8: Compose and wire `RouteVisualEditor`**

Render global Card, route-rule section Card, then rule-set section Card. Every list operation updates the complete route object through the route model functions.

- [ ] **Step 9: Verify and commit route editor**

```bash
npm --prefix ui test -- --run src/features/policy/route-form-model.test.ts src/features/policy/route-editor.test.tsx src/features/policy/policy-state.test.tsx
npm --prefix ui run typecheck
npm --prefix ui run lint
git add ui/src/features/policy/route-*.ts ui/src/features/policy/route-*.tsx ui/src/features/policy/policy-state.test.tsx
git commit -m "feat(route): add card form editor"
```

Expected: all checks and the commit succeed.

### Task 5: Model DNS globals, servers, rules, and actions

**Files:**

- Create: `ui/src/features/policy/dns-form-model.ts`
- Create: `ui/src/features/policy/dns-form-model.test.ts`

**Interfaces:**

```ts
export const dnsGlobalFields: readonly PolicyFieldSpec[]
export const legacyFakeIPFields: readonly PolicyFieldSpec[]
export const dnsServerTypes: readonly string[]
export const dnsRuleMatchFields: readonly PolicyFieldSpec[]
export const dnsActions: readonly string[]
export const dnsActionFields: Record<string, readonly PolicyFieldSpec[]>
export function dnsServers(object: JsonObject): JsonObject[]
export function dnsRules(object: JsonObject): JsonObject[]
export function setDNSServers(object: JsonObject, servers: readonly JsonObject[]): JsonObject
export function setDNSRules(object: JsonObject, rules: readonly JsonObject[]): JsonObject
export function inferDNSServerType(server: JsonObject): string
export function changeDNSServerType(server: JsonObject, type: string): JsonObject
export function changeDNSRuleType(rule: JsonObject, type: string): JsonObject
export function changeDNSAction(rule: JsonObject, action: string): JsonObject
export function summarizeDNSServer(server: JsonObject): { type: string; detail: string }
export function summarizeDNSRule(rule: JsonObject): { matches: string[]; action: string }
```

- [ ] **Step 1: Write failing global/FakeIP metadata tests**

Require `final`, strategy, cache switches/capacity, client subnet, reverse mapping, and nested legacy FakeIP enabled/IPv4/IPv6 fields.

- [ ] **Step 2: Write failing server inference and transition tests**

```ts
expect(inferDNSServerType({ address: "https://dns.google/dns-query" })).toBe("legacy")
expect(inferDNSServerType({ type: "https", server: "dns.google" })).toBe("https")
expect(changeDNSServerType({ address: "local", detour: "direct", custom: "keep" }, "udp"))
  .toEqual({ type: "udp", custom: "keep" })
expect(changeDNSServerType({ type: "https", server: "dns.google", path: "/dns-query", headers: { X: "1" }, tls: { enabled: true }, custom: "keep" }, "tls"))
  .toEqual({ type: "tls", server: "dns.google", tls: { enabled: true }, custom: "keep" })
```

Untouched legacy servers must preserve exact address-form JSON. Conversion occurs only after explicit Select changes.

- [ ] **Step 3: Write failing DNS rule tests**

```ts
export const dnsActions = ["route", "route-options", "reject", "predefined"] as const
```

Cover default/logical transitions, route server/strategy/cache/TTL/client subnet, reject fields, predefined response fields, summary generation, unknown retention, and target-driven cleanup.

- [ ] **Step 4: Run model tests and verify RED**

```bash
npm --prefix ui test -- --run src/features/policy/dns-form-model.test.ts
```

Expected: FAIL because the DNS model does not exist.

- [ ] **Step 5: Implement explicit DNS metadata**

```ts
export const dnsServerTypes = ["legacy", "local", "hosts", "udp", "tcp", "tls", "quic", "https", "h3", "dhcp", "fakeip"] as const
```

Keep Legacy metadata separate from modern type metadata. Validate list element types and preserve existing unknown server/action types as current Select values.

- [ ] **Step 6: Verify DNS model**

```bash
npm --prefix ui test -- --run src/features/policy/dns-form-model.test.ts
```

Expected: PASS.

---

### Task 6: Build DNS cards and dialogs

**Files:**

- Create: `ui/src/features/policy/dns-global-card.tsx`
- Create: `ui/src/features/policy/dns-server-card.tsx`
- Create: `ui/src/features/policy/dns-server-dialog.tsx`
- Create: `ui/src/features/policy/dns-rule-card.tsx`
- Create: `ui/src/features/policy/dns-rule-dialog.tsx`
- Create: `ui/src/features/policy/dns-visual-editor.tsx`
- Create: `ui/src/features/policy/dns-editor.test.tsx`
- Modify: `ui/src/features/policy/dns-page.tsx`

**Interfaces:**

```ts
export function DNSVisualEditor(props: PolicyVisualEditorProps): React.ReactNode
```

- [ ] **Step 1: Write failing global/FakeIP tests**

Require `final`, `strategy`, `disable_cache`, `disable_expire`, `independent_cache`, `cache_capacity`, `client_subnet`, `reverse_mapping`, FakeIP enabled/IPv4/IPv6 ranges, unknown retention, and pruning of an empty known-only `fakeip` object without deleting unknown FakeIP keys.

- [ ] **Step 2: Write failing server workflows**

Cover Empty/Add, untouched default-installer legacy servers, dynamic local/hosts/udp/tcp/tls/quic/https/h3/dhcp/fakeip fields, HTTPS headers validation, TLS and dialer visibility, summaries, edit/copy/delete, required fields, and unknown types through Advanced JSON.

- [ ] **Step 3: Write failing DNS rule workflows**

Cover these match fields:

```text
type, inbound, ip_version, query_type, network, auth_user, protocol,
domain, domain_suffix, domain_keyword, domain_regex,
source_ip_cidr, source_ip_is_private, ip_cidr, ip_is_private,
source_port, source_port_range, port, port_range,
process_name, process_path, process_path_regex, package_name,
user, user_id, outbound, clash_mode, rule_set,
rule_set_ip_cidr_match_source, network_type,
network_is_expensive, network_is_constrained, wifi_ssid, wifi_bssid, invert
```

Also cover route/route-options/reject/predefined actions, DNS server target Select, logical mode/invert, Advanced JSON child rules, edit/copy/delete/up/down, boundary disabled states, and invalid JSON blocking confirmation across tabs.

- [ ] **Step 4: Run DNS UI tests and verify RED**

```bash
npm --prefix ui test -- --run src/features/policy/dns-editor.test.tsx
```

Expected: FAIL because the DNS UI does not exist.

- [ ] **Step 5: Implement global and FakeIP Cards**

Use full Card composition. Do not create `fakeip` until a known field is set. Clearing all known fields prunes the object only when no unknown keys remain.

- [ ] **Step 6: Implement DNS server cards and Dialog**

Use Tabs:

```text
基础 | 拨号与解析 | TLS 与 HTTP | 类型专属 | 高级 JSON
```

Use stable Base UI Select items, `sm:max-w-5xl`, required title, internal scroll, and mounted structured JSON panels. Legacy fields change only after explicit server-type selection.

- [ ] **Step 7: Implement DNS rule cards and Dialog**

Reuse the interaction pattern, not route metadata. Use DNS server tags for route targets. Direct logical child rules use the shared structured list; nested logical children retain the JSON-array field.

- [ ] **Step 8: Compose and wire `DNSVisualEditor`**

Render DNS global Card, legacy FakeIP Card, server section Card, and DNS rule section Card. Every list operation updates the complete DNS object.

- [ ] **Step 9: Verify and commit DNS editor**

```bash
npm --prefix ui test -- --run src/features/policy/dns-form-model.test.ts src/features/policy/dns-editor.test.tsx src/features/policy/policy-state.test.tsx
npm --prefix ui run typecheck
npm --prefix ui run lint
git add ui/src/features/policy/dns-*.ts ui/src/features/policy/dns-*.tsx ui/src/features/policy/policy-state.test.tsx
git commit -m "feat(dns): add card form editor"
```

Expected: all checks and the commit succeed.

### Task 7: Add copy, mobile coverage, and integration regression

**Files:**

- Modify: `ui/src/i18n/locales/zh.ts`
- Modify: `ui/src/i18n/locales/en.ts`
- Modify: `ui/src/features/policy/policy-pages.test.tsx`
- Modify: `ui/e2e/app.spec.ts`
- Modify: `README.md`

**Interfaces:**

- Consumes all route/DNS editors from Tasks 1–6.
- Produces complete bilingual labels, 320px acceptance coverage, and concise documentation.

- [ ] **Step 1: Write failing translation interaction tests**

Assert Chinese labels for visual/advanced tabs, section Cards, CRUD, movement, Dialog tabs, actions, validation, and logical-rule guidance. Switch to English in one test and assert primary route/DNS labels resolve without raw keys.

- [ ] **Step 2: Add failing 320px Playwright tests**

Extend the mocked config with representative route rules/rule sets and DNS servers/rules. At 320px:

```ts
expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true)
```

Open a route-rule Dialog and DNS-server Dialog, assert their bounding boxes stay inside the viewport, and verify action menus/Tabs are keyboard reachable.

- [ ] **Step 3: Add concise bilingual copy**

Use `policy.route.*` and `policy.dns.*`. Keep existing save/install/rollback keys compatible. Labels must describe user concepts rather than Go type names.

- [ ] **Step 4: Update README**

Change only the existing traffic-policy bullet to mention card/form editing for route rules, rule sets, DNS servers, and DNS rules with Advanced JSON fallback.

- [ ] **Step 5: Verify and commit integration**

```bash
npm --prefix ui test -- --run src/features/policy
cd ui && npm run e2e
cd ..
git add README.md ui/src/i18n/locales/zh.ts ui/src/i18n/locales/en.ts ui/src/features/policy ui/e2e/app.spec.ts
git commit -m "test(policy): cover form editor flows"
```

Expected: policy tests, Playwright, and commit pass.

---

### Task 8: Review, verify, deploy, and close

**Files:**

- Review every file changed by Tasks 1–7.
- Modify only files required to fix Critical or Important findings.

- [ ] **Step 1: Request read-only code review**

Review sing-box field names/value shapes, legacy DNS preservation, target-driven cleanup, unknown retention, ordering/copy/delete, Advanced JSON revision, shadcn Base UI composition, accessibility, 320px layout, and size/complexity limits.

- [ ] **Step 2: Fix findings with TDD**

For each Critical or Important finding: add the smallest failing regression test, confirm RED, implement one fix, and rerun the targeted policy suite. Do not expand unrelated scope.

- [ ] **Step 3: Run frontend completion gates**

```bash
make check-ui
```

Expected: TypeScript, ESLint, all Vitest tests, all four coverage metrics at `>=90%`, and production build pass.

- [ ] **Step 4: Run Go formatting and completion gates**

```bash
goimports-reviser -rm-unused -set-alias -project-name github.com/xuthus5/boxd -recursive ./internal
goimports-reviser -rm-unused -set-alias -project-name github.com/xuthus5/boxd -recursive ./cmd
git diff --name-only -- '*.go'
make check-go
```

Expected: no Go files change; tests, race, Go coverage, golangci-lint, and govulncheck pass.

- [ ] **Step 5: Run final E2E and clean artifacts**

```bash
cd ui && npm run e2e
cd ..
rm -rf ui/coverage ui/playwright-report ui/test-results
git diff --check
git status --short
```

Expected: E2E passes and only intentional task files are modified.

- [ ] **Step 6: Commit review fixes if needed**

```bash
git add README.md ui/src/features/policy ui/src/i18n/locales/zh.ts ui/src/i18n/locales/en.ts ui/e2e/app.spec.ts
git commit -m "fix(policy): address editor review"
```

Skip this step when review produces no file changes; never create an empty commit.

- [ ] **Step 7: Build committed binary and verify embedded UI**

```bash
make build
make check-embedded-ui
```

Expected: `bin/boxd` contains the committed frontend and embedded-resource tests pass.

- [ ] **Step 8: Install and restart locally**

```bash
install -o root -g boxd -m 0750 bin/boxd /usr/local/bin/boxd.new
mv -f /usr/local/bin/boxd.new /usr/local/bin/boxd
systemctl restart boxd.service
```

- [ ] **Step 9: Verify deployment evidence**

```bash
cmp bin/boxd /usr/local/bin/boxd
systemctl is-active boxd.service
pid=$(systemctl show boxd.service -p MainPID --value)
readlink -f "/proc/$pid/exe"
curl -fsS -o /dev/null -w '%{http_code}\n' http://127.0.0.1:9090/
stat -c '%U:%G %a' /usr/local/bin/boxd
journalctl -u boxd.service --since '3 minutes ago' --no-pager -n 40
git status --short
```

Expected:

```text
active
/usr/local/bin/boxd
200
root:boxd 750
```

The journal contains no startup failure, panic, or fatal error, the deployed binary matches `bin/boxd`, and the Git worktree is clean.
