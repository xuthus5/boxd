import { ApiError } from "@/lib/api/client"

export type SubscriptionRefreshCode =
  | "invalid_url"
  | "network"
  | "timeout"
  | "unauthorized"
  | "forbidden"
  | "http_status"
  | "empty_content"
  | "not_found"
  | "unknown"

const HINT_KEYS: Record<string, string> = {
  invalid_url: "subscriptions.errorHintInvalidURL",
  network: "subscriptions.errorHintNetwork",
  timeout: "subscriptions.errorHintTimeout",
  unauthorized: "subscriptions.errorHintUnauthorized",
  forbidden: "subscriptions.errorHintForbidden",
  http_status: "subscriptions.errorHintHTTP",
  empty_content: "subscriptions.errorHintEmpty",
  not_found: "subscriptions.errorHintNotFound",
  unknown: "subscriptions.errorHintUnknown",
}

export function subscriptionErrorHintKey(code?: string): string {
  if (!code) return HINT_KEYS.unknown
  return HINT_KEYS[code] ?? HINT_KEYS.unknown
}

export function classifySubscriptionErrorMessage(message?: string): SubscriptionRefreshCode {
  const lower = (message ?? "").toLowerCase()
  if (!lower) return "unknown"
  if (lower.includes("subscription http 401") || lower.includes("unauthorized")) return "unauthorized"
  if (lower.includes("subscription http 403") || lower.includes("forbidden")) return "forbidden"
  if (lower.includes("subscription http")) return "http_status"
  if (lower.includes("no nodes") || lower.includes("empty")) return "empty_content"
  if (lower.includes("timeout") || lower.includes("deadline")) return "timeout"
  if (lower.includes("connection refused") || lower.includes("no such host") || lower.includes("network")) return "network"
  if (lower.includes("unsupported protocol") || lower.includes("invalid url") || lower.includes("://bad")) return "invalid_url"
  if (lower.includes("not found")) return "not_found"
  return "unknown"
}

export function resolveSubscriptionErrorCode(item: { error?: string; error_code?: string }): string | undefined {
  if (!item.error && !item.error_code) return undefined
  return item.error_code || classifySubscriptionErrorMessage(item.error)
}

export function classifySubscriptionRequestError(error: unknown): SubscriptionRefreshCode {
  if (error instanceof ApiError) {
    const code = error.code?.toLowerCase() || ""
    if (code === "invalid_request") return "invalid_url"
    if (code === "subscription_not_found" || code === "not_found") return "not_found"
    if (code === "subscription_refresh_failed") {
      return classifySubscriptionErrorMessage(error.message)
    }
    if (code === "unavailable") return "network"
    if (code === "timeout") return "timeout"
  }
  const message = error instanceof Error ? error.message : String(error || "")
  return classifySubscriptionErrorMessage(message)
}

export function subscriptionRequestErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message.trim()) return error.message.trim()
  return fallback
}

export function formatSubscriptionRequestErrorToast(error: unknown, fallback: string): string {
  const message = subscriptionRequestErrorMessage(error, fallback)
  const code = classifySubscriptionRequestError(error)
  if (!code || code === "unknown") return message
  return `${code}: ${message}`
}

export function subscriptionRequestErrorClipboardText(
  error: unknown,
  options: { scope?: string; id?: string; name?: string } = {},
): string {
  const message = subscriptionRequestErrorMessage(error, "")
  if (!message) return ""
  const code = classifySubscriptionRequestError(error)
  return [
    options.scope ? `scope: ${options.scope}` : "",
    options.id?.trim() ? `id: ${options.id.trim()}` : "",
    options.name?.trim() ? `name: ${options.name.trim()}` : "",
    code ? `code: ${code}` : "",
    `error: ${message}`,
  ].filter(Boolean).join("\n")
}

export type SubscriptionRefreshFailure = {
  id?: string
  name?: string
  code?: string
  message?: string
}

export type SubscriptionRefreshBatchSummary = {
  failed: number
  failedSamples: Array<{ name: string; code?: string; message: string }>
}

const FAILED_SAMPLE_LIMIT = 3

export function summarizeSubscriptionRefreshFailures(
  failures: readonly SubscriptionRefreshFailure[] | undefined,
): SubscriptionRefreshBatchSummary {
  const list = failures ?? []
  const failedSamples: SubscriptionRefreshBatchSummary["failedSamples"] = []
  for (const item of list) {
    if (failedSamples.length >= FAILED_SAMPLE_LIMIT) break
    const message = item.message?.trim() || "failed"
    const code = resolveSubscriptionErrorCode({ error: message, error_code: item.code })
    failedSamples.push({
      name: item.name?.trim() || item.id?.trim() || "—",
      code: code && code !== "unknown" ? code : undefined,
      message,
    })
  }
  return { failed: list.length, failedSamples }
}

export function formatSubscriptionRefreshBatchMessage(
  summary: SubscriptionRefreshBatchSummary,
  t: (key: string, values?: Record<string, string | number>) => string,
): string {
  if (summary.failed <= 0) return t("subscriptions.partialFailure")
  const parts = [t("subscriptions.partialFailureCount", { count: summary.failed })]
  if (summary.failedSamples.length) {
    const samples = summary.failedSamples
      .map((item) => (item.code ? `${item.name}: ${item.code}: ${item.message}` : `${item.name}: ${item.message}`))
      .join("; ")
    parts.push(t("subscriptions.refreshFailedSamples", { samples }))
  }
  return parts.join(" · ")
}

export function subscriptionRefreshBatchClipboardText(summary: SubscriptionRefreshBatchSummary): string {
  if (!summary.failedSamples.length) return ""
  return summary.failedSamples.map((sample) => ([
    `name: ${sample.name}`,
    sample.code ? `code: ${sample.code}` : "",
    `error: ${sample.message}`,
  ].filter(Boolean).join("\n"))).join("\n---\n")
}

export function extractSubscriptionRefreshFailures(data: unknown): SubscriptionRefreshFailure[] {
  if (!data || typeof data !== "object") return []
  const failed = (data as { failed?: unknown }).failed
  if (!Array.isArray(failed)) return []
  const rows: SubscriptionRefreshFailure[] = []
  for (const item of failed) {
    if (!item || typeof item !== "object") continue
    const row = item as Record<string, unknown>
    rows.push({
      id: typeof row.id === "string" ? row.id : undefined,
      name: typeof row.name === "string" ? row.name : undefined,
      code: typeof row.code === "string" ? row.code : undefined,
      message: typeof row.message === "string" ? row.message : undefined,
    })
  }
  return rows
}
