import { describe, expect, it, vi } from "vitest"

import { ApiError } from "@/lib/api/client"
import type { ConfigDiagnostics } from "@/lib/api/types"
import {
  buildSupportBundle,
  buildSupportBundleFilename,
  collectSupportBundle,
  countUnavailableSources,
  formatSupportBundle,
  SUPPORT_BUNDLE_SOURCES,
  type SupportBundleLoaders,
  type SupportBundleRequests,
  type SupportBundleSources,
} from "@/features/settings/support-bundle"

const exportedAt = new Date("2026-07-26T01:02:03.004Z")
const preferences = { theme: "dark", language: "zh", minimumLogLevel: "warn" } as const

function allRequests(status: "ok" | "unavailable" = "ok"): SupportBundleRequests {
  return Object.fromEntries(SUPPORT_BUNDLE_SOURCES.map((source) => [source, { status }])) as SupportBundleRequests
}

function fullSources(): SupportBundleSources {
  return {
    preferences,
    version: { version: "panel-1", kernel_version: "1.13.14" },
    service: {
      running: true,
      uptime: "2h",
      memory: 2048,
      version: "kernel-1",
      started_at: "2026-07-26T00:00:00Z",
      config_path: "/private/sing-box.json",
      last_error: "password=do-not-export",
      last_error_code: "restart_failed",
    },
    readiness: { status: "ready" },
    memory: { alloc: 10, total: 20, sys: 30, num_gc: 4, heap_inuse: 5, stack_inuse: 6, num_goroutine: 7 },
    config_diagnostics: {
      status: "warning",
      checked_at: "2026-07-26T00:01:00Z",
      summary: { errors: 1, warnings: 2 },
      counts: { inbounds: 1, outbounds: 2, endpoints: 1, route_rules: 3, rule_sets: 4, dns_servers: 2, dns_rules: 1 },
      features: { tun: true, clash_api: false, cache_file: true, fakeip: false, selector: true, urltest: false, wireguard: true, remote_rule_set: true },
      issues: [{ code: "unknown_outbound_reference", severity: "error", path: "route.rules[0]", value: "secret-node", detail: "token=do-not-export" }],
    } satisfies ConfigDiagnostics,
    rule_sets: [
      { tag: "ads", type: "local", format: "binary", path: "/private/ads.db", url: "https://secret.invalid/ads", last_etag: "secret-etag", builtin: true, updatable: true, last_updated: "2026-07-26T00:00:00Z" },
      { tag: "custom", type: "remote", builtin: false, updatable: false, last_updated: " " },
    ],
    rule_set_auto_update: { enabled: true, interval: "12h" },
    subscriptions: [
      { id: "sub-1", name: "private provider", url: "https://secret.invalid/sub", interval_min: 60, last_updated: "now", error: "private error", error_code: "timeout", traffic: { upload: 1, download: 2, total: 3 } },
      { id: "sub-2", name: "ok provider", url: "https://secret.invalid/ok", interval_min: 60, last_updated: "now" },
    ],
    nodes: [
      { tag: "secret-node", type: "vless", server: "secret.invalid", port: 443, source: "subscription", raw: { uuid: "secret-uuid" } },
      { tag: "direct", type: "direct", source: "import" },
    ],
    node_history: {
      history: {
        "secret-node": { tcp: [{ timestamp: "1", success: true, latency_ms: 10 }, { timestamp: "2", success: false }] },
        "direct": { http: [{ timestamp: "3", success: true, latency_ms: 30 }] },
      },
    },
    apply_history: {
      events: [
        { id: "private-id", source: "raw", status: "rolled_back", hash: "private-hash", size: 20, error: "token=private", error_code: "restart_failed", applied_at: "2026-07-26T00:00:00Z" },
        { id: "public-id", source: "update", status: "applied", hash: "public-hash", size: 10, applied_at: "2026-07-26T01:00:00Z" },
      ],
    },
    network: { interfaces: [{ name: "eth0", ips: ["10.0.0.2", "2001:db8::1"] }, { name: "lo", ips: [] }] },
  }
}

function successfulLoaders(): SupportBundleLoaders {
  return {
    version: async () => ({ version: "panel", kernel_version: "kernel" }),
    service: async () => ({ running: true }),
    readiness: async () => ({ status: "ready" }),
    memory: async () => ({ alloc: 1, total: 2, sys: 3, num_gc: 4, heap_inuse: 5, stack_inuse: 6 }),
    config_diagnostics: async () => fullSources().config_diagnostics!,
    rule_sets: async () => fullSources().rule_sets!,
    rule_set_auto_update: async () => ({ enabled: false, interval: "24h" }),
    subscriptions: async () => [],
    nodes: async () => [],
    node_history: async () => ({ history: {} }),
    apply_history: async () => ({ events: [] }),
    network: async () => ({ interfaces: [] }),
  }
}

describe("support bundle", () => {
  it("summarizes health signals while omitting sensitive payloads", () => {
    const bundle = buildSupportBundle(fullSources(), allRequests(), exportedAt)
    const serialized = formatSupportBundle(bundle)

    expect(bundle).toMatchObject({
      format_version: 1,
      product: "boxd",
      exported_at: exportedAt.toISOString(),
      panel: { version: "panel-1", kernel_version: "1.13.14", ready: true },
      service: { available: true, running: true, last_error_present: true, last_error_code: "restart_failed" },
      runtime: { available: true, memory: { alloc: 10, num_goroutine: 7 } },
      config: {
        diagnostics: { available: true, status: "warning", issue_codes: { unknown_outbound_reference: 1 }, issue_severities: { error: 1 } },
        rule_sets: { total: 2, builtin: 1, updatable: 1, with_last_updated: 1, types: { local: 1, remote: 1 } },
        apply_history: { count: 2, failure_count: 1, latest_applied_at: "2026-07-26T01:00:00Z" },
      },
      resources: {
        subscriptions: { total: 2, failed: 1, with_traffic: 1, error_codes: { timeout: 1 } },
        nodes: { total: 2, probe_samples: 3, probe_successes: 2, probe_failures: 1, average_latency_ms: 20 },
      },
      network: { interfaces: 2, ip_addresses: 2, ipv4_addresses: 1, ipv6_addresses: 1 },
    })
    for (const secret of [
      "do-not-export", "/private/sing-box.json", "secret-node", "secret.invalid", "secret-uuid", "private-id", "private-hash", "token=private",
    ]) expect(serialized).not.toContain(secret)
    expect(bundle.redaction.strategy).toBe("allowlist")
    expect(serialized.endsWith("\n")).toBe(true)
  })

  it("marks unavailable sections without inventing data", () => {
    const bundle = buildSupportBundle({ preferences }, allRequests("unavailable"), exportedAt)

    expect(bundle.panel).toMatchObject({ version: null, kernel_version: null, ready: null })
    expect(bundle.service.available).toBe(false)
    expect(bundle.runtime).toEqual({ available: false, memory: null })
    expect(bundle.config.diagnostics).toMatchObject({ available: false, status: null, summary: null, counts: null, features: null, issue_codes: {} })
    expect(bundle.config.rule_sets).toMatchObject({ available: false, total: 0, auto_update: { available: false } })
    expect(bundle.config.apply_history).toMatchObject({ available: false, count: 0, failure_count: 0 })
    expect(bundle.resources.subscriptions).toMatchObject({ available: false, total: 0 })
    expect(bundle.resources.nodes).toMatchObject({ available: false, total: 0, probe_samples: 0, average_latency_ms: null })
    expect(bundle.network).toEqual({ available: false, interfaces: 0, ip_addresses: 0, ipv4_addresses: 0, ipv6_addresses: 0 })
  })

  it("ignores malformed list entries instead of throwing", () => {
    const malformed = {
      preferences,
      rule_sets: [null],
      subscriptions: [null],
      nodes: [null],
      apply_history: { events: [null, { status: "applied", applied_at: "now" }] },
      node_history: { history: { malformed: { tcp: [{ success: true }] } } },
      network: { interfaces: [null, {}, { ips: ["127.0.0.1", 42] }] },
    } as unknown as SupportBundleSources

    const bundle = buildSupportBundle(malformed, allRequests(), exportedAt)

    expect(bundle.config.rule_sets.total).toBe(0)
    expect(bundle.config.apply_history.count).toBe(1)
    expect(bundle.resources.subscriptions.failed).toBe(0)
    expect(bundle.network).toMatchObject({ interfaces: 2, ip_addresses: 1, ipv4_addresses: 1, ipv6_addresses: 0 })
  })

  it("collects every source concurrently and preserves partial failures", async () => {
    const loaders = successfulLoaders()
    loaders.service = vi.fn().mockRejectedValue(new ApiError("private message", 503, "service_unavailable"))
    loaders.network = vi.fn().mockRejectedValue(new Error("network secret"))
    loaders.memory = () => { throw new ApiError("sync failure", 500, "memory_failed") }

    const bundle = await collectSupportBundle(loaders, preferences, exportedAt)

    expect(bundle.requests.service).toEqual({ status: "unavailable", error_code: "service_unavailable" })
    expect(bundle.requests.network).toEqual({ status: "unavailable", error_code: "unavailable" })
    expect(bundle.requests.version).toEqual({ status: "ok" })
    expect(bundle.panel.version).toBe("panel")
    expect(bundle.requests.memory).toEqual({ status: "unavailable", error_code: "memory_failed" })
    expect(countUnavailableSources(bundle.requests)).toBe(3)
    expect(Object.values(loaders).every((loader) => vi.isMockFunction(loader) || typeof loader === "function")).toBe(true)
  })

  it("formats a stable filename and counts successful requests", () => {
    expect(buildSupportBundleFilename(exportedAt)).toBe("boxd-support-bundle-2026-07-26T01-02-03-004Z.json")
    expect(countUnavailableSources(allRequests())).toBe(0)
  })
})
