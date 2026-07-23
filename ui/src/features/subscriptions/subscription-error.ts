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
  if (!item.error) return undefined
  return item.error_code || classifySubscriptionErrorMessage(item.error)
}
