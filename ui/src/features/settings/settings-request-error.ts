import { ApiError } from "@/lib/api/client"

export type SettingsRequestErrorCode =
  | "unauthorized"
  | "invalid_input"
  | "unavailable"
  | "network"
  | "timeout"
  | "internal"
  | "unknown"

const HINT_KEYS: Record<SettingsRequestErrorCode, string> = {
  unauthorized: "settings.errorHintUnauthorized",
  invalid_input: "settings.errorHintInvalid",
  unavailable: "settings.errorHintUnavailable",
  network: "settings.errorHintNetwork",
  timeout: "settings.errorHintTimeout",
  internal: "settings.errorHintInternal",
  unknown: "settings.errorHintUnknown",
}

export function settingsRequestErrorHintKey(code?: string): string {
  if (!code) return HINT_KEYS.unknown
  return HINT_KEYS[code as SettingsRequestErrorCode] ?? HINT_KEYS.unknown
}

export function classifySettingsRequestError(error: unknown): SettingsRequestErrorCode {
  if (error instanceof ApiError) {
    const code = error.code?.toLowerCase() || ""
    if (code === "unauthorized" || error.status === 401) return "unauthorized"
    if (code === "invalid_request" || code === "invalid_response" || error.status === 400) return "invalid_input"
    if (code === "unavailable" || error.status === 503) return "unavailable"
    if (code === "timeout" || error.status === 408 || error.status === 504) return "timeout"
    if (code === "internal_error" || error.status >= 500) return "internal"
    if (code === "bad_gateway" || code === "request_failed") return "network"
  }
  const message = error instanceof Error ? error.message : String(error || "")
  const lower = message.toLowerCase()
  if (!lower) return "unknown"
  if (lower.includes("unauthorized") || lower.includes("current password") || lower.includes("wrong password")) {
    return "unauthorized"
  }
  if (
    lower.includes("invalid")
    || lower.includes("required")
    || lower.includes("must be")
    || lower.includes("too short")
    || lower.includes("too long")
    || lower.includes("parse")
  ) {
    return "invalid_input"
  }
  if (lower.includes("timeout") || lower.includes("deadline")) return "timeout"
  if (lower.includes("not available") || lower.includes("unavailable")) return "unavailable"
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

export function settingsRequestErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message.trim()) return error.message.trim()
  return fallback
}

export function formatSettingsRequestErrorToast(error: unknown, fallback: string): string {
  const message = settingsRequestErrorMessage(error, fallback)
  const code = classifySettingsRequestError(error)
  if (!code || code === "unknown") return message
  return `${code}: ${message}`
}

export function settingsRequestErrorClipboardText(
  error: unknown,
  options: { scope?: string; field?: string } = {},
): string {
  const message = settingsRequestErrorMessage(error, "")
  if (!message) return ""
  const code = classifySettingsRequestError(error)
  return [
    options.scope ? `scope: ${options.scope}` : "",
    options.field?.trim() ? `field: ${options.field.trim()}` : "",
    code ? `code: ${code}` : "",
    `error: ${message}`,
  ].filter(Boolean).join("\n")
}
