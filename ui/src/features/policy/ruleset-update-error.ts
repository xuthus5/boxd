import type { RuleSetUpdateResult, RuleSetUpdateResponse } from "@/lib/api/types"

export type RuleSetUpdateErrorCode =
  | "not_updatable"
  | "unsupported"
  | "invalid_url"
  | "network"
  | "timeout"
  | "http_status"
  | "empty_content"
  | "permission"
  | "cache"
  | "unknown"

const HINT_KEYS: Record<string, string> = {
  not_updatable: "policy.route.errorHintNotUpdatable",
  unsupported: "policy.route.errorHintUnsupported",
  invalid_url: "policy.route.errorHintInvalidURL",
  network: "policy.route.errorHintNetwork",
  timeout: "policy.route.errorHintTimeout",
  http_status: "policy.route.errorHintHTTP",
  empty_content: "policy.route.errorHintEmpty",
  permission: "policy.route.errorHintPermission",
  cache: "policy.route.errorHintCache",
  unknown: "policy.route.errorHintUnknown",
}

export function ruleSetErrorHintKey(code?: string): string {
  if (!code) return HINT_KEYS.unknown
  return HINT_KEYS[code] ?? HINT_KEYS.unknown
}

export function classifyRuleSetErrorMessage(message?: string): RuleSetUpdateErrorCode {
  const lower = (message ?? "").toLowerCase()
  if (!lower) return "unknown"
  if (lower.includes("not updatable")) return "not_updatable"
  if (lower.includes("not auto-updated") || lower.includes("not supported")) return "unsupported"
  if (lower.includes("url is empty") || lower.includes("invalid url") || lower.includes("unsupported protocol")) {
    return "invalid_url"
  }
  if (lower.includes("empty rule-set") || lower.includes("empty body") || lower.includes("empty content")) {
    return "empty_content"
  }
  if (lower.includes("unexpected status")) return "http_status"
  if (lower.includes("permission denied") || lower.includes("operation not permitted")) return "permission"
  if (lower.includes("cache is unavailable") || lower.includes("bbolt")) return "cache"
  if (lower.includes("timeout") || lower.includes("deadline") || lower.includes("i/o timeout")) return "timeout"
  if (
    lower.includes("connection refused")
    || lower.includes("connection reset")
    || lower.includes("no such host")
    || lower.includes("network")
    || lower.includes("tls:")
    || lower.includes("x509")
  ) return "network"
  return "unknown"
}

export function resolveRuleSetErrorCode(result: {
  ok?: boolean
  error?: string
  error_code?: string
}): string | undefined {
  if (result.ok) return undefined
  if (!result.error && !result.error_code) return undefined
  return result.error_code || classifyRuleSetErrorMessage(result.error)
}

export type RuleSetFailureSample = {
  tag: string
  error: string
  code?: string
}

export type RuleSetUpdateSummary = {
  updated: number
  failed: number
  skipped: number
  restarted: boolean
  failedSamples: RuleSetFailureSample[]
}

const FAILED_SAMPLE_LIMIT = 3

export function summarizeRuleSetUpdate(response?: Partial<RuleSetUpdateResponse> | null): RuleSetUpdateSummary {
  const results = response?.results ?? []
  const failedSamples: RuleSetFailureSample[] = []
  for (const item of results) {
    if (item.ok) continue
    const code = resolveRuleSetErrorCode(item)
    if (code === "not_updatable" || code === "unsupported") continue
    if (failedSamples.length >= FAILED_SAMPLE_LIMIT) continue
    failedSamples.push({
      tag: item.tag?.trim() || "—",
      error: item.error?.trim() || "failed",
      code: code && code !== "unknown" ? code : undefined,
    })
  }
  return {
    updated: response?.updated_count ?? 0,
    failed: response?.failed_count ?? 0,
    skipped: response?.skipped_count ?? 0,
    restarted: Boolean(response?.restarted),
    failedSamples,
  }
}

export function formatRuleSetFailureSample(sample: RuleSetFailureSample): string {
  if (sample.code && sample.code !== sample.error) return `${sample.tag}: ${sample.code}: ${sample.error}`
  return `${sample.tag}: ${sample.error}`
}

export function formatRuleSetUpdateMessage(
  summary: RuleSetUpdateSummary,
  t: (key: string, values?: Record<string, string | number>) => string,
): string {
  if (summary.failed === 0 && summary.updated === 0 && summary.skipped === 0) {
    return t("policy.route.ruleSetUpdateSuccess", { updated: 0 })
  }
  const parts: string[] = []
  if (summary.failed > 0 && summary.updated > 0) {
    parts.push(t("policy.route.ruleSetUpdatePartial", {
      updated: summary.updated,
      failed: summary.failed,
    }))
  } else if (summary.failed > 0) {
    parts.push(t("policy.route.ruleSetUpdateFailed", { failed: summary.failed }))
  } else {
    parts.push(t("policy.route.ruleSetUpdateSuccess", { updated: summary.updated }))
  }
  if (summary.skipped > 0) {
    parts.push(t("policy.route.ruleSetUpdateSkipped", { skipped: summary.skipped }))
  }
  if (summary.restarted) {
    parts.push(t("policy.route.ruleSetUpdateRestarted"))
  }
  if (summary.failedSamples.length) {
    const samples = summary.failedSamples.map(formatRuleSetFailureSample).join("; ")
    parts.push(t("policy.route.ruleSetUpdateFailedSamples", { samples }))
  }
  return parts.join(" · ")
}

export function ruleSetUpdateToastTone(summary: RuleSetUpdateSummary): "success" | "warning" | "error" {
  if (summary.failed > 0 && summary.updated === 0) return "error"
  if (summary.failed > 0) return "warning"
  return "success"
}

export function ruleSetUpdateErrorClipboardText(result: Pick<
  RuleSetUpdateResult,
  "tag" | "type" | "error" | "error_code" | "ok"
>): string {
  if (result.ok) return ""
  const code = resolveRuleSetErrorCode(result)
  const lines = [
    result.tag?.trim() ? `tag: ${result.tag.trim()}` : "",
    result.type?.trim() ? `type: ${result.type.trim()}` : "",
    code ? `code: ${code}` : "",
    result.error?.trim() ? `error: ${result.error.trim()}` : "",
  ].filter(Boolean)
  return lines.join("\n")
}

export function classifyRuleSetRequestError(error: unknown): RuleSetUpdateErrorCode {
  if (error && typeof error === "object" && "code" in error) {
    const code = String((error as { code?: string }).code || "").toLowerCase()
    if (code in HINT_KEYS) return code as RuleSetUpdateErrorCode
    if (code === "unavailable") return "network"
    if (code === "timeout") return "timeout"
  }
  const message = error instanceof Error ? error.message : String(error || "")
  return classifyRuleSetErrorMessage(message)
}

export function ruleSetRequestErrorClipboardText(error: unknown, scope = "update"): string {
  const message = error instanceof Error ? error.message.trim() : String(error || "").trim()
  if (!message) return ""
  const code = classifyRuleSetRequestError(error)
  return [`scope: ${scope}`, `code: ${code}`, `error: ${message}`].join("\n")
}

export function formatRuleSetRequestErrorToast(
  error: unknown,
  _t: (key: string, values?: Record<string, string | number>) => string,
  fallback: string,
): string {
  const message = error instanceof Error && error.message.trim() ? error.message.trim() : fallback
  const code = classifyRuleSetRequestError(error)
  if (!code || code === "unknown") return message
  return `${code}: ${message}`
}

export function ruleSetBatchFailureClipboardText(summary: RuleSetUpdateSummary): string {
  if (!summary.failedSamples.length) return ""
  return summary.failedSamples.map((sample) => {
    const code = sample.code ? `code: ${sample.code}` : ""
    return [ `tag: ${sample.tag}`, code, `error: ${sample.error}` ].filter(Boolean).join("\n")
  }).join("\n---\n")
}

