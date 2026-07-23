/** SSE / live stream failure diagnostics for logs and connections. */

export type StreamErrorCode =
  | "unauthorized"
  | "unavailable"
  | "network"
  | "invalid_payload"
  | "timeout"
  | "unknown"

export function classifyStreamErrorMessage(message?: string): StreamErrorCode {
  const lower = (message ?? "").toLowerCase()
  if (!lower) return "unknown"
  if (lower.includes("401") || lower.includes("unauthorized")) return "unauthorized"
  if (lower.includes("503") || lower.includes("unavailable") || lower.includes("service not available")) {
    return "unavailable"
  }
  if (lower.includes("invalid sse") || lower.includes("invalid json") || lower.includes("event data")) {
    return "invalid_payload"
  }
  if (lower.includes("timeout") || lower.includes("deadline") || lower.includes("aborted")) return "timeout"
  if (
    lower.includes("failed to fetch")
    || lower.includes("network")
    || lower.includes("offline")
    || lower.includes("connection")
    || lower.includes("status 5")
    || lower.includes("status 0")
  ) return "network"
  if (lower.includes("status 4")) return "unavailable"
  return "unknown"
}

export function streamErrorHintKey(code?: string): string {
  switch (code) {
    case "unauthorized":
      return "observability.errorHintStreamUnauthorized"
    case "unavailable":
      return "observability.errorHintStreamUnavailable"
    case "network":
      return "observability.errorHintStreamNetwork"
    case "invalid_payload":
      return "observability.errorHintStreamInvalidPayload"
    case "timeout":
      return "observability.errorHintStreamTimeout"
    default:
      return "observability.errorHintStreamUnknown"
  }
}

export function streamErrorClipboardText(options: {
  path?: string
  status?: string
  paused?: boolean
  error?: string
  code?: string
}): string {
  const error = options.error?.trim()
  if (!error) return ""
  const code = options.code || classifyStreamErrorMessage(error)
  const lines = [
    options.path?.trim() ? `path: ${options.path.trim()}` : "",
    options.status?.trim() ? `status: ${options.status.trim()}` : "",
    options.paused === undefined ? "" : `paused: ${options.paused ? "true" : "false"}`,
    code ? `code: ${code}` : "",
    `error: ${error}`,
  ].filter(Boolean)
  return lines.join("\n")
}

export function formatStreamErrorTitle(
  t: (key: string, values?: Record<string, string | number>) => string,
  code?: string,
): string {
  if (!code || code === "unknown") return t("observability.streamError")
  return t("observability.streamErrorWithCode", { code })
}
