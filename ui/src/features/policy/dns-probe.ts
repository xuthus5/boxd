import type { JsonObject } from "@/features/policy/policy-form-model"
import type { DNSProbeInput, DNSProbeResult } from "@/lib/api/types"

const nonProbeable = new Set(["local", "hosts", "dhcp", "fakeip", "tailscale"])
const FAILED_SAMPLE_LIMIT = 3

export type DNSProbeFailureSample = {
  tag: string
  error: string
}

export type DNSProbeBatchSummary = {
  total: number
  success: number
  failed: number
  avgLatencyMs?: number
  bestTag?: string
  bestLatencyMs?: number
  worstTag?: string
  worstLatencyMs?: number
  failedSamples: DNSProbeFailureSample[]
}

export function isDNSServerProbeable(item: JsonObject): boolean {
  const type = typeof item.type === "string" ? item.type.toLowerCase() : ""
  if (type && nonProbeable.has(type)) return false
  if (typeof item.server === "string" && item.server.trim()) return true
  if (typeof item.address === "string" && item.address.trim()) return true
  return false
}

export function dnsProbeInput(item: JsonObject, domain?: string): DNSProbeInput | null {
  if (!isDNSServerProbeable(item)) return null
  const input: DNSProbeInput = {}
  if (typeof item.tag === "string" && item.tag) input.tag = item.tag
  if (typeof item.type === "string" && item.type) input.type = item.type
  if (typeof item.server === "string" && item.server) input.server = item.server
  if (typeof item.server_port === "number" && Number.isFinite(item.server_port)) {
    input.server_port = item.server_port
  }
  if (typeof item.address === "string" && item.address) input.address = item.address
  if (typeof item.path === "string" && item.path) input.path = item.path
  if (domain) input.domain = domain
  return input
}

function sampleTag(result: DNSProbeResult, fallback: string): string {
  const tag = result.tag?.trim()
  if (tag) return tag
  return fallback
}

function trackLatency(
  label: string,
  latency: number | undefined,
  state: { sum: number; count: number; bestTag?: string; best?: number; worstTag?: string; worst?: number },
) {
  if (typeof latency !== "number" || !Number.isFinite(latency)) return
  state.sum += latency
  state.count += 1
  if (state.best === undefined || latency < state.best) {
    state.best = latency
    state.bestTag = label
  }
  if (state.worst === undefined || latency > state.worst) {
    state.worst = latency
    state.worstTag = label
  }
}

export function summarizeDNSProbeResults(
  results: readonly DNSProbeResult[] | undefined,
): DNSProbeBatchSummary {
  const list = results ?? []
  let success = 0
  let failed = 0
  const latency = { sum: 0, count: 0 } as {
    sum: number; count: number; bestTag?: string; best?: number; worstTag?: string; worst?: number
  }
  const failedSamples: DNSProbeFailureSample[] = []
  list.forEach((item, index) => {
    const label = sampleTag(item, `#${index + 1}`)
    if (item.success) {
      success += 1
      trackLatency(label, item.latency_ms, latency)
      return
    }
    failed += 1
    if (failedSamples.length < FAILED_SAMPLE_LIMIT) {
      failedSamples.push({ tag: label, error: item.error?.trim() || "failed" })
    }
  })
  return {
    total: list.length,
    success,
    failed,
    avgLatencyMs: latency.count ? Math.round((latency.sum / latency.count) * 10) / 10 : undefined,
    bestTag: latency.bestTag,
    bestLatencyMs: latency.best,
    worstTag: latency.worstTag,
    worstLatencyMs: latency.worst,
    failedSamples,
  }
}

export function formatDNSProbeBatchMessage(
  summary: DNSProbeBatchSummary,
  t: (key: string, values?: Record<string, string | number>) => string,
): string {
  if (summary.total === 0) return t("policy.dns.probeBatchComplete")
  const parts = [
    t("policy.dns.probeBatchSummary", {
      success: summary.success,
      failed: summary.failed,
      total: summary.total,
      avg: summary.avgLatencyMs === undefined ? "—" : `${summary.avgLatencyMs}ms`,
    }),
  ]
  if (summary.bestTag && summary.bestLatencyMs !== undefined) {
    parts.push(t("policy.dns.probeBatchBest", {
      tag: summary.bestTag,
      latency: `${summary.bestLatencyMs}ms`,
    }))
  }
  if (summary.worstTag && summary.worstLatencyMs !== undefined) {
    parts.push(t("policy.dns.probeBatchWorst", {
      tag: summary.worstTag,
      latency: `${summary.worstLatencyMs}ms`,
    }))
  }
  if (summary.failedSamples.length) {
    const samples = summary.failedSamples
      .map((item) => `${item.tag}: ${item.error}`)
      .join("; ")
    parts.push(t("policy.dns.probeBatchFailedSamples", { samples }))
  }
  return parts.join(" · ")
}

export function dnsProbeBatchToastTone(
  summary: DNSProbeBatchSummary,
): "success" | "warning" | "error" {
  if (summary.failed > 0 && summary.success === 0) return "error"
  if (summary.failed > 0) return "warning"
  return "success"
}

/** 将批量探测结果按输入顺序写回卡片索引，并保留 tag 索引。 */
export function mapDNSProbeBatchResults(
  inputs: readonly DNSProbeInput[],
  results: readonly DNSProbeResult[],
  probeKeyForInput: (input: DNSProbeInput, index: number) => string,
): Record<string, DNSProbeResult> {
  const next: Record<string, DNSProbeResult> = {}
  results.forEach((result, index) => {
    const input = inputs[index]
    const key = input ? probeKeyForInput(input, index) : `idx:${index}`
    next[key] = result
    if (result.tag?.trim()) next[result.tag.trim()] = result
    else if (input?.tag?.trim()) next[input.tag.trim()] = result
  })
  return next
}
