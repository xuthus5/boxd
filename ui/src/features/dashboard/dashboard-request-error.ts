import { ApiError } from "@/lib/api/client"
import {
  classifyKernelErrorMessage,
  kernelErrorHintKey,
  type KernelErrorCode,
} from "@/features/dashboard/kernel-error"

export type DashboardRequestErrorCode =
  | KernelErrorCode
  | "unavailable"
  | "network"
  | "timeout"
  | "internal"

const EXTRA_HINT_KEYS: Record<"unavailable" | "network" | "timeout" | "internal", string> = {
  unavailable: "dashboard.errorHintRequestUnavailable",
  network: "dashboard.errorHintRequestNetwork",
  timeout: "dashboard.errorHintRequestTimeout",
  internal: "dashboard.errorHintRequestInternal",
}

export function dashboardRequestErrorHintKey(code?: string): string {
  if (!code) return kernelErrorHintKey("unknown")
  if (code in EXTRA_HINT_KEYS) return EXTRA_HINT_KEYS[code as keyof typeof EXTRA_HINT_KEYS]
  return kernelErrorHintKey(code)
}

export function classifyDashboardRequestError(error: unknown): DashboardRequestErrorCode {
  if (error instanceof ApiError) {
    const code = error.code?.toLowerCase() || ""
    if (code === "unavailable" || error.status === 503) return "unavailable"
    if (code === "timeout" || error.status === 408 || error.status === 504) return "timeout"
    if (code === "bad_gateway" || code === "request_failed") return "network"
    if (code === "internal_error" || error.status >= 500) {
      const fromMessage = classifyKernelErrorMessage(error.message)
      return fromMessage === "unknown" ? "internal" : fromMessage
    }
  }
  const message = error instanceof Error ? error.message : String(error || "")
  const lower = message.toLowerCase()
  if (!lower) return "unknown"
  const fromMessage = classifyKernelErrorMessage(message)
  if (fromMessage !== "unknown") return fromMessage
  if (lower.includes("timeout") || lower.includes("deadline")) return "timeout"
  if (lower.includes("not available") || lower.includes("unavailable") || lower.includes("not running")) {
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

export function dashboardRequestErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message.trim()) return error.message.trim()
  return fallback
}

export function formatDashboardRequestErrorToast(error: unknown, fallback: string): string {
  const message = dashboardRequestErrorMessage(error, fallback)
  const code = classifyDashboardRequestError(error)
  if (!code || code === "unknown") return message
  return `${code}: ${message}`
}

export function dashboardRequestErrorClipboardText(
  error: unknown,
  options: { scope?: string; action?: string } = {},
): string {
  const message = dashboardRequestErrorMessage(error, "")
  if (!message) return ""
  const code = classifyDashboardRequestError(error)
  return [
    options.scope ? `scope: ${options.scope}` : "",
    options.action?.trim() ? `action: ${options.action.trim()}` : "",
    code ? `code: ${code}` : "",
    `error: ${message}`,
  ].filter(Boolean).join("\n")
}
