import { ApiError } from "@/lib/api/client"

/** Stable codes for page query load failures. */
export type PageLoadErrorCode =
  | "unauthorized"
  | "unavailable"
  | "network"
  | "timeout"
  | "internal"
  | "unknown"

const HINT_KEYS: Record<PageLoadErrorCode, string> = {
  unauthorized: "common.errorHintUnauthorized",
  unavailable: "common.errorHintUnavailable",
  network: "common.errorHintNetwork",
  timeout: "common.errorHintTimeout",
  internal: "common.errorHintInternal",
  unknown: "common.errorHintUnknown",
}

export function pageLoadErrorHintKey(code?: string): string {
  if (!code) return HINT_KEYS.unknown
  return HINT_KEYS[code as PageLoadErrorCode] ?? HINT_KEYS.unknown
}

export function classifyPageLoadError(error: unknown): PageLoadErrorCode {
  if (error instanceof ApiError) {
    const code = error.code?.toLowerCase() || ""
    if (code === "unauthorized" || error.status === 401) return "unauthorized"
    if (code === "unavailable" || error.status === 503) return "unavailable"
    if (code === "timeout" || error.status === 408 || error.status === 504) return "timeout"
    if (code === "bad_gateway" || code === "request_failed") return "network"
    if (code === "internal_error" || error.status >= 500) return "internal"
  }
  const message = error instanceof Error ? error.message : String(error || "")
  const lower = message.toLowerCase()
  if (!lower) return "unknown"
  if (lower.includes("unauthorized") || lower.includes("401")) return "unauthorized"
  if (lower.includes("timeout") || lower.includes("deadline")) return "timeout"
  if (lower.includes("not available") || lower.includes("unavailable") || lower.includes("503")) {
    return "unavailable"
  }
  if (
    lower.includes("failed to fetch")
    || lower.includes("network")
    || lower.includes("connection refused")
    || lower.includes("offline")
  ) {
    return "network"
  }
  if (lower.includes("internal") || lower.includes("failed to")) return "internal"
  return "unknown"
}

export function pageLoadErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message.trim()) return error.message.trim()
  if (typeof error === "string" && error.trim()) return error.trim()
  return fallback
}

export function formatPageLoadErrorTitle(
  t: (key: string, values?: Record<string, string | number>) => string,
  code: PageLoadErrorCode,
  titleKey = "common.loadFailed",
): string {
  if (!code || code === "unknown") return t(titleKey)
  if (titleKey === "common.loadFailed") return t("common.loadFailedWithCode", { code })
  return `${t(titleKey)} (${code})`
}

export function pageLoadErrorClipboardText(
  error: unknown,
  options: { scope?: string; path?: string } = {},
): string {
  const message = pageLoadErrorMessage(error, "")
  if (!message) return ""
  const code = classifyPageLoadError(error)
  return [
    options.scope?.trim() ? `scope: ${options.scope.trim()}` : "",
    options.path?.trim() ? `path: ${options.path.trim()}` : "",
    code ? `code: ${code}` : "",
    `error: ${message}`,
  ].filter(Boolean).join("\n")
}
