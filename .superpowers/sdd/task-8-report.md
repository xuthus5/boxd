# Task 8 Report: Policy Final Review Fixes

## Outcome

- Route action transitions now preserve `network_type` as a matcher while still removing direct-only fields.
- Route/DNS page boundaries reject non-array list roots and non-object list entries without normalizing or filtering the source JSON.
- Invalid section structures disable Save, render a destructive shadcn Alert in Visual mode, and remain editable in Advanced JSON.
- Route Rule、Route RuleSet、DNS Rule 与 DNS Server 共享 raw JSON、last-valid object、revision、invalid-field 与 CodeMirror synchronization 状态。
- Route/DNS numeric fields now enforce sing-box-specific integer, range, list, mark, and IP-version constraints through the existing transform invalid channel.
- Route/DNS logical rules require at least one object child; DNS predefined actions no longer incorrectly require `rcode`.

## Review Finding RED / GREEN Evidence

### Critical: Route matcher cleanup

- RED: Route model and UI tests both showed `network_type: ["wifi"]` disappearing when `direct` changed to `reject`.
- GREEN: `knownActionFields` excludes paths also owned by `routeMatchFields`; direct-only `bind_interface` and `routing_mark` are still removed.
- Verification: 2 files / 25 tests passed.

### Important 1: Section array structure

- RED: 3/3 tests failed because no shared validator or Visual Alert existed and `renderVisual` still received filtered data.
- GREEN: `isPolicySectionStructureValid` validates Route `rules`/`rule_set` and DNS `servers`/`rules`; undefined is allowed, present values must be all-object arrays.
- Advanced JSON recovery tests preserve unknown object payloads and re-enable Visual mode and Save only after the structure is valid.
- Verification with page/model regressions: 4 files / 65 tests passed.

### Important 2: Route dialog last-valid state

- RED: Route Rule and RuleSet lost all Tabs after Advanced JSON became invalid; all 4 resilience tests failed.
- GREEN: `usePolicyDialogState` keeps raw JSON and the last valid object separately. Tabs remain mounted, invalid raw text survives tab changes, Save stays disabled, and a structured edit replaces invalid raw JSON and refreshes CodeMirror.
- DNS retains `useDNSDialogState` as a thin wrapper around the shared hook.
- Verification: Route/DNS dialog state and validation, 3 files / 42 tests passed; TypeScript passed.

### Important 3: Numeric validation

- RED: the new helper module did not exist and 4 UI validation tests failed because fields had no invalid state or Save gate.
- GREEN: `policy-number-transform.ts` provides path-specific integer, integer-list, and FwMark transforms. Route and DNS compose their own registries without changing `PolicyFieldSpec`.
- Verification: numeric unit/UI, existing DNS transforms, and Route state, 4 files / 20 tests passed; TypeScript passed.

### Important 4/5: Logical rules and predefined DNS

- RED: 8 failures showed Route/DNS accepting `[]` and `[1]`, and DNS rejecting predefined actions without `rcode`.
- GREEN: shared `isNonEmptyJsonObjectArray` requires `rules.length > 0 && every(isJsonObject)`; `[{}]` is accepted at the UI shape boundary. DNS predefined `rcode` is optional and remains transform-validated when present.
- Verification with Route/DNS editors: 5 files / 65 tests passed.

## sing-box v1.13.14 Schema Check

- `option.LogicalRule.IsValid()` and `option.LogicalDNSRule.IsValid()` require `len(Rules) > 0` and valid child rules; their JSON fields are `[]Rule` and `[]DNSRule`.
- The UI intentionally implements the requested shape gate only: non-empty and every entry an object. Backend validation remains authoritative for child semantics.
- `DNSRouteActionPredefined.Rcode` is `*DNSRCode` with `omitempty`; `DNSRCode.Build()` returns `dns.RcodeSuccess` when nil. Empty predefined actions therefore represent an empty NOERROR response and are valid.
- Route/DNS ports are `uint16`; cache capacity, rewrite TTL, and route-options fallback delay are `uint32`; user IDs are `int32`; FwMark is `uint32` with number-or-string JSON support.

## Numeric Constraint Matrix

- Route/DNS `source_port`, `port`: integer lists, `0..65535`.
- DNS `server_port`, Route `override_port`: integer, `0..65535`.
- DNS `cache_capacity`, DNS/Route `rewrite_ttl`, nested `domain_resolver.rewrite_ttl`, Route `default_domain_resolver.rewrite_ttl`, numeric route-options `fallback_delay`: integer, `0..4294967295`.
- Route/DNS `user_id`: nonnegative sing-box `int32`, `0..2147483647`.
- Route `default_mark`/`routing_mark`, DNS `routing_mark`: decimal input writes a number; valid `0x`/`0o`/`0b` input remains a string; maximum `uint32`.
- Route/DNS `ip_version`: only `4` or `6`.
- Empty input deletes the field. Negative, fractional, `NaN`, `Infinity`, and overflow values return `null`, render `data-invalid`/`aria-invalid`, and block Save.
- Duration text fields remain unconstrained.

## Final Verification

- Policy tests: 20 files / 220 tests passed.
- TypeScript: passed.
- ESLint: passed.
- Full coverage: 60 files / 373 tests passed.
  - Statements: 96.00% (2212/2304)
  - Branches: 91.69% (1435/1565)
  - Functions: 93.64% (825/881)
  - Lines: 97.59% (1705/1747)
- Full Playwright: 4/4 passed.
- Production UI build: passed; 2986 modules transformed.
- `git diff --check`: clean.
- Temporary `ui/coverage`, `ui/playwright-report`, `ui/test-results`, and `ui/dist` outputs were removed.

## Size, Scope, and Composition Audit

- Largest Policy test file: `dns-editor.test.tsx`, 300 lines.
- Largest changed production file: `dns-form-model.ts`, 296 lines.
- New production files are at most 62 lines; changed Dialog files remain at most 166 lines.
- shadcn Alert, FieldGroup, Dialog Title, TabsList/Trigger, and Base Select grouping are retained.
- No backend/API, dependency, global CSS, README, Go, installer, or deployment changes were made.

## Concerns

- No known functional blockers or remaining Critical/Important findings.
- Go gates and local deployment were intentionally not run because the parent task explicitly reserved them for the main integration thread.
- Playwright emitted the existing `NO_COLOR`/`FORCE_COLOR` warning; all tests passed and it did not affect behavior.
