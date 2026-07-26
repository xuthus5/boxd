import type { ConfigApplyEvent, LatencyPoint, NetworkInterfaceInfo, Outbound, RuleSetStatusItem, Subscription } from "@/lib/api/types"

import type { SupportBundle, SupportBundleSources } from "@/features/settings/support-bundle"

const OMITTED_FIELDS = [
  "passwords and JWT secrets",
  "authentication tokens",
  "subscription URLs and names",
  "node tags, server addresses, and raw node settings",
  "raw sing-box configuration",
  "log messages and connection targets",
  "filesystem paths",
  "rule-set URLs, paths, and ETags",
]

function safeText(value: unknown): string | null {
  if (typeof value !== "string") return null
  const trimmed = value.trim()
  return trimmed ? trimmed.slice(0, 160) : null
}

function safeNumber(value: unknown): number | null {
  return typeof value === "number" && Number.isFinite(value) ? value : null
}

function safeBoolean(value: unknown): boolean | null {
  return typeof value === "boolean" ? value : null
}

function safeToken(value: unknown): string | null {
  const text = safeText(value)
  if (!text) return null
  const token = text.replace(/[^a-zA-Z0-9_.:/-]/g, "_").slice(0, 64)
  return token || null
}

function countValues(values: readonly unknown[]): Record<string, number> {
  const counts: Record<string, number> = {}
  for (const value of values) {
    const token = safeToken(value)
    if (token) counts[token] = (counts[token] ?? 0) + 1
  }
  return Object.fromEntries(Object.entries(counts).sort(([left], [right]) => left.localeCompare(right)))
}

function summaryFromNumbers(value: Record<string, unknown> | undefined, keys: readonly string[]) {
  if (!value) return null
  return Object.fromEntries(keys.map((key) => [key, safeNumber(value[key])]))
}

function summaryFromBooleans(value: Record<string, unknown> | undefined, keys: readonly string[]) {
  if (!value) return null
  return Object.fromEntries(keys.map((key) => [key, safeBoolean(value[key])]))
}

function objectItems<T>(value: unknown): T[] {
  return Array.isArray(value)
    ? value.filter((item): item is T => Boolean(item && typeof item === "object"))
    : []
}

function summarizePanel(sources: SupportBundleSources): SupportBundle["panel"] {
  return {
    version: safeText(sources.version?.version),
    kernel_version: safeText(sources.version?.kernel_version),
    ready: sources.readiness ? sources.readiness.status === "ready" : null,
    preferences: {
      theme: safeToken(sources.preferences.theme),
      language: safeToken(sources.preferences.language),
      minimum_log_level: safeToken(sources.preferences.minimumLogLevel),
    },
  }
}

function summarizeService(sources: SupportBundleSources): SupportBundle["service"] {
  const service = sources.service
  return {
    available: Boolean(service),
    running: safeBoolean(service?.running),
    uptime: safeText(service?.uptime),
    memory_bytes: safeNumber(service?.memory),
    version: safeText(service?.version),
    started_at: safeText(service?.started_at),
    last_error_present: service ? Boolean(safeText(service.last_error)) : null,
    last_error_code: safeToken(service?.last_error_code),
  }
}

function summarizeRuntime(sources: SupportBundleSources): SupportBundle["runtime"] {
  const memory = sources.memory
  if (!memory) return { available: false, memory: null }
  return {
    available: true,
    memory: {
      alloc: safeNumber(memory.alloc),
      total: safeNumber(memory.total),
      sys: safeNumber(memory.sys),
      num_gc: safeNumber(memory.num_gc),
      heap_inuse: safeNumber(memory.heap_inuse),
      stack_inuse: safeNumber(memory.stack_inuse),
      num_goroutine: safeNumber(memory.num_goroutine),
    },
  }
}

function summarizeDiagnostics(sources: SupportBundleSources): SupportBundle["config"]["diagnostics"] {
  const diagnostics = sources.config_diagnostics
  const issues = objectItems<{ code?: unknown; severity?: unknown }>(diagnostics?.issues)
  const counts = diagnostics?.counts as unknown as Record<string, unknown> | undefined
  const summary = diagnostics?.summary as unknown as Record<string, unknown> | undefined
  const features = diagnostics?.features as unknown as Record<string, unknown> | undefined
  return {
    available: Boolean(diagnostics),
    status: safeToken(diagnostics?.status),
    checked_at: safeText(diagnostics?.checked_at),
    summary: summaryFromNumbers(summary, ["errors", "warnings"]),
    counts: summaryFromNumbers(counts, ["inbounds", "outbounds", "endpoints", "route_rules", "rule_sets", "dns_servers", "dns_rules"]),
    features: summaryFromBooleans(features, ["tun", "clash_api", "cache_file", "fakeip", "selector", "urltest", "wireguard", "remote_rule_set"]),
    issue_codes: countValues(issues.map((issue) => issue?.code)),
    issue_severities: countValues(issues.map((issue) => issue?.severity)),
  }
}

function summarizeRuleSets(sources: SupportBundleSources): SupportBundle["config"]["rule_sets"] {
  const items = objectItems<RuleSetStatusItem>(sources.rule_sets)
  const autoUpdate = sources.rule_set_auto_update
  return {
    available: Array.isArray(sources.rule_sets),
    total: items.length,
    builtin: items.filter((item) => item.builtin).length,
    updatable: items.filter((item) => item.updatable).length,
    with_last_updated: items.filter((item) => Boolean(safeText(item.last_updated))).length,
    types: countValues(items.map((item) => item.type)),
    formats: countValues(items.map((item) => item.format)),
    auto_update: {
      available: Boolean(autoUpdate),
      enabled: safeBoolean(autoUpdate?.enabled),
      interval: safeText(autoUpdate?.interval),
    },
  }
}

function isApplyFailure(status: unknown): boolean {
  return status === "rolled_back" || status === "validate_failed"
}

function summarizeApplyHistory(sources: SupportBundleSources): SupportBundle["config"]["apply_history"] {
  const events = objectItems<ConfigApplyEvent>(sources.apply_history?.events)
  const latest = events
    .map((event) => safeText(event.applied_at))
    .filter((value): value is string => Boolean(value))
    .sort()
    .at(-1) ?? null
  return {
    available: Array.isArray(sources.apply_history?.events),
    count: events.length,
    failure_count: events.filter((event) => isApplyFailure(event.status)).length,
    statuses: countValues(events.map((event) => event.status)),
    error_codes: countValues(events.map((event) => event.error_code)),
    latest_applied_at: latest,
  }
}

function summarizeSubscriptions(sources: SupportBundleSources): SupportBundle["resources"]["subscriptions"] {
  const items = objectItems<Subscription>(sources.subscriptions)
  return {
    available: Array.isArray(sources.subscriptions),
    total: items.length,
    failed: items.filter((item) => Boolean(safeText(item.error))).length,
    with_traffic: items.filter((item) => Boolean(item.traffic)).length,
    error_codes: countValues(items.map((item) => item.error_code)),
  }
}

function collectLatencyPoints(value: unknown): LatencyPoint[] {
  const points: LatencyPoint[] = []
  const pending: unknown[] = [value]
  while (pending.length > 0 && points.length < 10000) {
    const current = pending.pop()
    if (Array.isArray(current)) {
      pending.push(...current)
      continue
    }
    if (!current || typeof current !== "object") continue
    const candidate = current as Partial<LatencyPoint>
    if (typeof candidate.success === "boolean") {
      points.push({ success: candidate.success, latency_ms: safeNumber(candidate.latency_ms) ?? undefined, timestamp: safeText(candidate.timestamp) ?? "" })
      continue
    }
    pending.push(...Object.values(current))
  }
  return points
}

function summarizeProbes(history: unknown) {
  const points = collectLatencyPoints(history)
  const latencies = points.map((point) => point.latency_ms).filter((value): value is number => typeof value === "number")
  const average = latencies.length > 0 ? latencies.reduce((sum, value) => sum + value, 0) / latencies.length : null
  return {
    probe_samples: points.length,
    probe_successes: points.filter((point) => point.success).length,
    probe_failures: points.filter((point) => !point.success).length,
    average_latency_ms: average === null ? null : Math.round(average * 100) / 100,
  }
}

function summarizeNodes(sources: SupportBundleSources): SupportBundle["resources"]["nodes"] {
  const items = objectItems<Outbound>(sources.nodes)
  return {
    available: Array.isArray(sources.nodes),
    total: items.length,
    types: countValues(items.map((item) => item.type)),
    sources: countValues(items.map((item) => item.source)),
    ...summarizeProbes(sources.node_history?.history),
  }
}

function summarizeNetwork(sources: SupportBundleSources): SupportBundle["network"] {
  const interfaces = objectItems<NetworkInterfaceInfo>(sources.network?.interfaces)
  let ipAddresses = 0
  let ipv4Addresses = 0
  let ipv6Addresses = 0
  for (const item of interfaces) {
    const addresses = Array.isArray(item.ips) ? item.ips : []
    for (const address of addresses) {
      if (typeof address !== "string") continue
      ipAddresses += 1
      if (address.includes(":")) ipv6Addresses += 1
      else ipv4Addresses += 1
    }
  }
  return {
    available: Array.isArray(sources.network?.interfaces),
    interfaces: interfaces.length,
    ip_addresses: ipAddresses,
    ipv4_addresses: ipv4Addresses,
    ipv6_addresses: ipv6Addresses,
  }
}

export function buildSupportBundle(
  sources: SupportBundleSources,
  requests: SupportBundle["requests"],
  exportedAt = new Date(),
): SupportBundle {
  return {
    format_version: 1,
    product: "boxd",
    exported_at: exportedAt.toISOString(),
    redaction: { strategy: "allowlist", omitted: [...OMITTED_FIELDS] },
    requests,
    panel: summarizePanel(sources),
    service: summarizeService(sources),
    runtime: summarizeRuntime(sources),
    config: {
      diagnostics: summarizeDiagnostics(sources),
      rule_sets: summarizeRuleSets(sources),
      apply_history: summarizeApplyHistory(sources),
    },
    resources: {
      subscriptions: summarizeSubscriptions(sources),
      nodes: summarizeNodes(sources),
    },
    network: summarizeNetwork(sources),
  }
}
