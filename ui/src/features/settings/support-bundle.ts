import { ApiError } from "@/lib/api/client"
import { buildSupportBundle } from "@/features/settings/support-bundle-summary"
import type {
  ConfigDiagnostics,
  MemoryStats,
  Outbound,
  ReadinessStatus,
  RuleSetAutoUpdate,
  RuleSetStatusItem,
  ServiceStatus,
  Subscription,
  UIPreferences,
  VersionInfo,
} from "@/lib/api/types"

export { buildSupportBundle }

export const SUPPORT_BUNDLE_SOURCES = [
  "version",
  "service",
  "readiness",
  "memory",
  "config_diagnostics",
  "rule_sets",
  "rule_set_auto_update",
  "subscriptions",
  "nodes",
  "node_history",
  "apply_history",
  "network",
] as const

export type SupportBundleSource = typeof SUPPORT_BUNDLE_SOURCES[number]

export type SupportBundleRequestState = {
  status: "ok" | "unavailable"
  error_code?: string
}

export type SupportBundleRequests = Record<SupportBundleSource, SupportBundleRequestState>

export interface SupportNodeHistoryPayload {
  history?: unknown
}

export interface SupportApplyHistoryPayload {
  events?: unknown
}

export interface SupportNetworkPayload {
  interfaces?: unknown
}

export interface SupportBundleLoaders {
  version: () => Promise<VersionInfo>
  service: () => Promise<ServiceStatus>
  readiness: () => Promise<ReadinessStatus>
  memory: () => Promise<MemoryStats>
  config_diagnostics: () => Promise<ConfigDiagnostics>
  rule_sets: () => Promise<RuleSetStatusItem[]>
  rule_set_auto_update: () => Promise<RuleSetAutoUpdate>
  subscriptions: () => Promise<Subscription[]>
  nodes: () => Promise<Outbound[]>
  node_history: () => Promise<SupportNodeHistoryPayload>
  apply_history: () => Promise<SupportApplyHistoryPayload>
  network: () => Promise<SupportNetworkPayload>
}

export interface SupportBundleSources {
  version?: VersionInfo
  service?: ServiceStatus
  readiness?: ReadinessStatus
  memory?: MemoryStats
  config_diagnostics?: ConfigDiagnostics
  rule_sets?: RuleSetStatusItem[]
  rule_set_auto_update?: RuleSetAutoUpdate
  subscriptions?: Subscription[]
  nodes?: Outbound[]
  node_history?: SupportNodeHistoryPayload
  apply_history?: SupportApplyHistoryPayload
  network?: SupportNetworkPayload
  preferences: UIPreferences
}

export interface SupportBundle {
  format_version: 1
  product: "boxd"
  exported_at: string
  redaction: {
    strategy: "allowlist"
    omitted: string[]
  }
  requests: SupportBundleRequests
  panel: {
    version: string | null
    kernel_version: string | null
    ready: boolean | null
    preferences: {
      theme: string | null
      language: string | null
      minimum_log_level: string | null
    }
  }
  service: {
    available: boolean
    running: boolean | null
    uptime: string | null
    memory_bytes: number | null
    version: string | null
    started_at: string | null
    last_error_present: boolean | null
    last_error_code: string | null
  }
  runtime: {
    available: boolean
    memory: Record<string, number | null> | null
  }
  config: {
    diagnostics: {
      available: boolean
      status: string | null
      checked_at: string | null
      summary: Record<string, number | null> | null
      counts: Record<string, number | null> | null
      features: Record<string, boolean | null> | null
      issue_codes: Record<string, number>
      issue_severities: Record<string, number>
    }
    rule_sets: {
      available: boolean
      total: number
      builtin: number
      updatable: number
      with_last_updated: number
      types: Record<string, number>
      formats: Record<string, number>
      auto_update: {
        available: boolean
        enabled: boolean | null
        interval: string | null
      }
    }
    apply_history: {
      available: boolean
      count: number
      failure_count: number
      statuses: Record<string, number>
      error_codes: Record<string, number>
      latest_applied_at: string | null
    }
  }
  resources: {
    subscriptions: {
      available: boolean
      total: number
      failed: number
      with_traffic: number
      error_codes: Record<string, number>
    }
    nodes: {
      available: boolean
      total: number
      types: Record<string, number>
      sources: Record<string, number>
      probe_samples: number
      probe_successes: number
      probe_failures: number
      average_latency_ms: number | null
    }
  }
  network: {
    available: boolean
    interfaces: number
    ip_addresses: number
    ipv4_addresses: number
    ipv6_addresses: number
  }
}

function errorCode(reason: unknown): string {
  if (reason instanceof ApiError && reason.code.trim()) {
    const code = reason.code.trim().replace(/[^a-zA-Z0-9_.:/-]/g, "_").slice(0, 64)
    return code || "unavailable"
  }
  return "unavailable"
}

function requestState(result: PromiseSettledResult<unknown>): SupportBundleRequestState {
  return result.status === "fulfilled"
    ? { status: "ok" }
    : { status: "unavailable", error_code: errorCode(result.reason) }
}

function resultValue<T>(result: PromiseSettledResult<T>): T | undefined {
  return result.status === "fulfilled" ? result.value : undefined
}

function invokeLoader<T>(loader: () => Promise<T>): Promise<T> {
  try {
    return Promise.resolve(loader())
  } catch (error) {
    return Promise.reject(error)
  }
}

export async function collectSupportBundle(
  loaders: SupportBundleLoaders,
  preferences: UIPreferences,
  exportedAt = new Date(),
): Promise<SupportBundle> {
  const results = await Promise.allSettled([
    invokeLoader(loaders.version), invokeLoader(loaders.service), invokeLoader(loaders.readiness), invokeLoader(loaders.memory), invokeLoader(loaders.config_diagnostics),
    invokeLoader(loaders.rule_sets), invokeLoader(loaders.rule_set_auto_update), invokeLoader(loaders.subscriptions), invokeLoader(loaders.nodes),
    invokeLoader(loaders.node_history), invokeLoader(loaders.apply_history), invokeLoader(loaders.network),
  ])
  const [version, service, readiness, memory, diagnostics, ruleSets, autoUpdate, subscriptions, nodes, nodeHistory, applyHistory, network] = results
  const sources: SupportBundleSources = {
    version: resultValue(version) as VersionInfo | undefined,
    service: resultValue(service) as ServiceStatus | undefined,
    readiness: resultValue(readiness) as ReadinessStatus | undefined,
    memory: resultValue(memory) as MemoryStats | undefined,
    config_diagnostics: resultValue(diagnostics) as ConfigDiagnostics | undefined,
    rule_sets: resultValue(ruleSets) as RuleSetStatusItem[] | undefined,
    rule_set_auto_update: resultValue(autoUpdate) as RuleSetAutoUpdate | undefined,
    subscriptions: resultValue(subscriptions) as Subscription[] | undefined,
    nodes: resultValue(nodes) as Outbound[] | undefined,
    node_history: resultValue(nodeHistory) as SupportNodeHistoryPayload | undefined,
    apply_history: resultValue(applyHistory) as SupportApplyHistoryPayload | undefined,
    network: resultValue(network) as SupportNetworkPayload | undefined,
    preferences,
  }
  const requests = Object.fromEntries(SUPPORT_BUNDLE_SOURCES.map((source, index) => [source, requestState(results[index])])) as SupportBundleRequests
  return buildSupportBundle(sources, requests, exportedAt)
}

export function formatSupportBundle(bundle: SupportBundle): string {
  return `${JSON.stringify(bundle, null, 2)}\n`
}

export function buildSupportBundleFilename(now = new Date()): string {
  const stamp = now.toISOString().replace(/[:.]/g, "-")
  return `boxd-support-bundle-${stamp}.json`
}

export function countUnavailableSources(requests: SupportBundleRequests): number {
  return Object.values(requests).filter((request) => request.status === "unavailable").length
}
