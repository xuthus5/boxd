import { ApiError } from "@/lib/api/client"

export type LoginErrorCode =
  | "unauthorized"
  | "rate_limited"
  | "invalid_input"
  | "network"
  | "timeout"
  | "internal"
  | "unknown"

export interface LoginErrorState {
  message: string
  code: LoginErrorCode
}

const HINT_KEYS: Record<LoginErrorCode, string> = {
  unauthorized: "auth.errorHintUnauthorized",
  rate_limited: "auth.errorHintRateLimited",
  invalid_input: "auth.errorHintInvalid",
  network: "auth.errorHintNetwork",
  timeout: "auth.errorHintTimeout",
  internal: "auth.errorHintInternal",
  unknown: "auth.errorHintUnknown",
}

export function loginErrorHintKey(code?: string): string {
  if (!code) return HINT_KEYS.unknown
  return HINT_KEYS[code as LoginErrorCode] ?? HINT_KEYS.unknown
}

export function classifyLoginError(error: unknown): LoginErrorCode {
  if (error instanceof ApiError) {
    const code = error.code?.toLowerCase() || ""
    if (code === "unauthorized" || error.status === 401) return "unauthorized"
    if (code === "rate_limited" || error.status === 429) return "rate_limited"
    if (code === "invalid_request" || code === "invalid_response" || error.status === 400) {
      return "invalid_input"
    }
    if (code === "timeout" || error.status === 408 || error.status === 504) return "timeout"
    if (code === "unavailable" || error.status === 503) return "network"
    if (code === "bad_gateway" || code === "request_failed") return "network"
    if (code === "internal_error" || error.status >= 500) return "internal"
  }
  const message = error instanceof Error ? error.message : String(error || "")
  const lower = message.toLowerCase()
  if (!lower) return "unknown"
  if (lower.includes("too many login") || lower.includes("rate limit") || lower.includes("too many")) {
    return "rate_limited"
  }
  if (lower.includes("invalid credentials") || lower.includes("unauthorized") || lower.includes("wrong password")) {
    return "unauthorized"
  }
  if (lower.includes("invalid request") || lower.includes("invalid json") || lower.includes("required")) {
    return "invalid_input"
  }
  if (lower.includes("timeout") || lower.includes("deadline")) return "timeout"
  if (
    lower.includes("failed to fetch")
    || lower.includes("network")
    || lower.includes("connection refused")
    || lower.includes("offline")
  ) {
    return "network"
  }
  if (lower.includes("internal") || lower.includes("token generation")) return "internal"
  return "unknown"
}

export function loginErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message.trim()) return error.message.trim()
  return fallback
}

export function loginErrorFromError(error: unknown, fallback: string): LoginErrorState {
  return {
    message: loginErrorMessage(error, fallback),
    code: classifyLoginError(error),
  }
}

export function formatLoginErrorTitle(
  t: (key: string, values?: Record<string, string | number>) => string,
  error: LoginErrorState,
): string {
  if (error.code && error.code !== "unknown") {
    return t("auth.failedWithCode", { code: error.code })
  }
  return t("auth.failed")
}

export function loginErrorClipboardText(error: LoginErrorState | null | undefined): string {
  if (!error?.message?.trim()) return ""
  return [
    error.code ? `code: ${error.code}` : "",
    `error: ${error.message.trim()}`,
  ].filter(Boolean).join("\n")
}
