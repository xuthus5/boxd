import { formatLatency } from "@/features/nodes/node-format"
import { ApiError } from "@/lib/api/client"
import { api } from "@/lib/api/endpoints"
import type { OutboundGroup } from "@/lib/api/types"

export type DelayFailure = {
  failed: true
  error: string
  code?: string
}

export type DelayValue = number | DelayFailure
export type DelayMap = Record<string, DelayValue>

export type DelayFailureSample = {
  tag: string
  error: string
  code?: string
}

export type DelayBatchSummary = {
  total: number
  ok: number
  failed: number
  avgLatencyMs?: number
  bestTag?: string
  bestLatencyMs?: number
  worstTag?: string
  worstLatencyMs?: number
  failedSamples: DelayFailureSample[]
}

const preferredTags = ["proxy", "select", "GLOBAL"]
const FAILED_SAMPLE_LIMIT = 3

export function isDelayFailure(value: DelayValue | undefined): value is DelayFailure {
  return Boolean(value && typeof value === "object" && value.failed)
}

export function pickPrimaryGroup(groups: OutboundGroup[]) {
  const selectors = groups.filter((group) => group.type === "selector" && group.all.length > 0)
  if (!selectors.length) return null
  for (const tag of preferredTags) {
    const found = selectors.find((group) => group.tag === tag)
    if (found) return found
  }
  return selectors[0]
}

export function formatDelayValue(value: DelayValue | undefined, failedLabel: string) {
  if (isDelayFailure(value)) {
    if (value.code && value.code !== "unknown" && value.code !== value.error) {
      return `${value.code}: ${value.error || failedLabel}`
    }
    return value.error || failedLabel
  }
  if (typeof value === "number") return formatLatency(value)
  return "—"
}

export function sortDelayEntries(delays: DelayMap) {
  return Object.entries(delays).sort((left, right) => {
    const leftOk = typeof left[1] === "number"
    const rightOk = typeof right[1] === "number"
    if (leftOk !== rightOk) return leftOk ? -1 : 1
    if (leftOk && rightOk && left[1] !== right[1]) return Number(left[1]) - Number(right[1])
    return left[0].localeCompare(right[0])
  })
}

export function classifyDelayErrorMessage(message?: string, code?: string): string {
  const fromCode = (code ?? "").toLowerCase()
  if (fromCode === "unavailable") return "unavailable"
  if (fromCode === "not_found" || fromCode === "runtime_group_not_found") return "not_found"
  if (fromCode === "runtime_not_selectable") return "unsupported"
  const lower = (message ?? "").toLowerCase()
  if (!lower && !fromCode) return "unknown"
  if (lower.includes("not running") || lower.includes("unavailable") || lower.includes("service not available")) {
    return "unavailable"
  }
  if (lower.includes("not found")) return "not_found"
  if (lower.includes("no response")) return "no_response"
  if (lower.includes("timeout") || lower.includes("deadline") || lower.includes("i/o timeout")) return "timeout"
  if (
    lower.includes("connection refused")
    || lower.includes("connection reset")
    || lower.includes("no such host")
    || lower.includes("network")
    || lower.includes("tls")
    || lower.includes("x509")
  ) return "network"
  if (fromCode === "runtime_delay_failed") return "timeout"
  return "unknown"
}

export function delayErrorHintKey(code?: string): string {
  switch (code) {
    case "unavailable":
      return "dashboard.errorHintDelayUnavailable"
    case "not_found":
      return "dashboard.errorHintDelayNotFound"
    case "no_response":
      return "dashboard.errorHintDelayNoResponse"
    case "timeout":
      return "dashboard.errorHintDelayTimeout"
    case "network":
      return "dashboard.errorHintDelayNetwork"
    case "unsupported":
      return "dashboard.errorHintDelayUnsupported"
    default:
      return "dashboard.errorHintDelayUnknown"
  }
}

export function delayFailureFromError(error: unknown): DelayFailure {
  if (error instanceof ApiError) {
    const message = error.message?.trim() || "delay test failed"
    return {
      failed: true,
      error: message,
      code: classifyDelayErrorMessage(message, error.code),
    }
  }
  if (error instanceof Error) {
    const message = error.message?.trim() || "delay test failed"
    return {
      failed: true,
      error: message,
      code: classifyDelayErrorMessage(message),
    }
  }
  return { failed: true, error: "delay test failed", code: "unknown" }
}

export function summarizeDelays(delays: DelayMap): DelayBatchSummary {
  let ok = 0
  let failed = 0
  let latencySum = 0
  let latencyCount = 0
  let bestTag: string | undefined
  let bestLatencyMs: number | undefined
  let worstTag: string | undefined
  let worstLatencyMs: number | undefined
  const failedSamples: DelayFailureSample[] = []

  for (const [tag, value] of Object.entries(delays)) {
    if (typeof value === "number" && Number.isFinite(value)) {
      ok += 1
      latencySum += value
      latencyCount += 1
      if (bestLatencyMs === undefined || value < bestLatencyMs) {
        bestLatencyMs = value
        bestTag = tag
      }
      if (worstLatencyMs === undefined || value > worstLatencyMs) {
        worstLatencyMs = value
        worstTag = tag
      }
      continue
    }
    failed += 1
    if (failedSamples.length < FAILED_SAMPLE_LIMIT) {
      const failure = isDelayFailure(value)
        ? value
        : { failed: true as const, error: "failed", code: "unknown" }
      failedSamples.push({
        tag: tag.trim() || "—",
        error: failure.error || "failed",
        code: failure.code && failure.code !== "unknown" ? failure.code : undefined,
      })
    }
  }

  return {
    total: ok + failed,
    ok,
    failed,
    avgLatencyMs: latencyCount ? Math.round((latencySum / latencyCount) * 10) / 10 : undefined,
    bestTag,
    bestLatencyMs,
    worstTag,
    worstLatencyMs,
    failedSamples,
  }
}

export function formatDelayFailureSample(sample: DelayFailureSample): string {
  if (sample.code && sample.code !== sample.error) return `${sample.tag}: ${sample.code}: ${sample.error}`
  return `${sample.tag}: ${sample.error}`
}

export function formatDelayBatchMessage(
  summary: DelayBatchSummary,
  t: (key: string, values?: Record<string, string | number>) => string,
): string {
  if (summary.total === 0) return t("dashboard.proxyDelayDone", { ok: 0, failed: 0, total: 0 })
  const parts = [
    t("dashboard.proxyDelayDone", {
      ok: summary.ok,
      failed: summary.failed,
      total: summary.total,
    }),
  ]
  if (summary.avgLatencyMs !== undefined) {
    parts.push(t("dashboard.proxyDelayAvg", { avg: `${summary.avgLatencyMs}ms` }))
  }
  if (summary.bestTag && summary.bestLatencyMs !== undefined) {
    parts.push(t("dashboard.proxyDelayBest", {
      tag: summary.bestTag,
      latency: `${summary.bestLatencyMs}ms`,
    }))
  }
  if (summary.worstTag && summary.worstLatencyMs !== undefined) {
    parts.push(t("dashboard.proxyDelayWorst", {
      tag: summary.worstTag,
      latency: `${summary.worstLatencyMs}ms`,
    }))
  }
  if (summary.failedSamples.length) {
    const samples = summary.failedSamples.map(formatDelayFailureSample).join("; ")
    parts.push(t("dashboard.proxyDelayFailedSamples", { samples }))
  }
  return parts.join(" · ")
}

export function delayBatchToastTone(summary: DelayBatchSummary): "success" | "warning" | "error" {
  if (summary.failed > 0 && summary.ok === 0) return "error"
  if (summary.failed > 0) return "warning"
  return "success"
}

export function delayFailureClipboardText(tag: string, value: DelayFailure): string {
  const lines = [
    tag.trim() ? `tag: ${tag.trim()}` : "",
    value.code ? `code: ${value.code}` : "",
    value.error?.trim() ? `error: ${value.error.trim()}` : "",
  ].filter(Boolean)
  return lines.join("\n")
}

async function probeMemberDelays(members: readonly string[]): Promise<DelayMap> {
  const entries = await Promise.all(members.map(async (tag) => {
    try {
      const result = await api.nodes.delay(tag) as { delay?: number }
      if (typeof result.delay === "number" && result.delay > 0) {
        return [tag, result.delay] as const
      }
      return [tag, {
        failed: true as const,
        error: "delay test failed: no response",
        code: "no_response",
      }] as const
    } catch (error) {
      return [tag, delayFailureFromError(error)] as const
    }
  }))
  return Object.fromEntries(entries)
}

export async function measureGroupDelays(groupTag: string, members: readonly string[]): Promise<DelayMap> {
  try {
    const urlTest = await api.nodes.urlTest(groupTag)
    const next: DelayMap = {}
    for (const tag of members) {
      const value = urlTest[tag]
      if (typeof value === "number" && Number.isFinite(value)) {
        next[tag] = value
      } else {
        next[tag] = {
          failed: true,
          error: "delay test failed: no response",
          code: "no_response",
        }
      }
    }
    return next
  } catch {
    return probeMemberDelays(members)
  }
}
