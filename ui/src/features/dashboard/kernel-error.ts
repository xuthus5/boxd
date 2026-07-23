/** Kernel start / config-apply stable error codes for ops diagnostics. */

export type KernelErrorCode =
  | "config_invalid"
  | "config_missing"
  | "restart_failed"
  | "start_failed"
  | "permission"
  | "unknown"

const HINT_KEYS: Record<string, string> = {
  config_invalid: "dashboard.errorHintConfigInvalid",
  config_missing: "dashboard.errorHintConfigMissing",
  restart_failed: "dashboard.errorHintRestartFailed",
  start_failed: "dashboard.errorHintStartFailed",
  permission: "dashboard.errorHintPermission",
  unknown: "dashboard.errorHintUnknown",
}

export function kernelErrorHintKey(code?: string): string {
  if (!code) return HINT_KEYS.unknown
  return HINT_KEYS[code] ?? HINT_KEYS.unknown
}

export function classifyKernelErrorMessage(message?: string): KernelErrorCode {
  const lower = (message ?? "").toLowerCase()
  if (!lower) return "unknown"
  if (lower.includes("no such file") || lower.includes("not exist") || lower.includes("cannot find")) {
    return "config_missing"
  }
  if (lower.includes("permission denied") || lower.includes("operation not permitted")) {
    return "permission"
  }
  if (lower.includes("restart failed")) return "restart_failed"
  if (
    lower.includes("decode")
    || lower.includes("invalid")
    || lower.includes("unmarshal")
    || lower.includes("unknown field")
    || lower.includes("missing required")
    || lower.includes("legacy")
    || lower.includes("required")
  ) {
    return "config_invalid"
  }
  if (
    lower.includes("start failed")
    || lower.includes("listen")
    || lower.includes("bind")
    || lower.includes("address already in use")
    || lower.includes("factory failed")
  ) {
    return "start_failed"
  }
  return "unknown"
}

export function resolveKernelErrorCode(item: {
  error?: string
  error_code?: string
  last_error?: string
  last_error_code?: string
}): string | undefined {
  const code = item.error_code?.trim() || item.last_error_code?.trim()
  const message = item.error?.trim() || item.last_error?.trim()
  if (!message && !code) return undefined
  return code || classifyKernelErrorMessage(message)
}

export function kernelLastErrorClipboardText(status: {
  running?: boolean
  config_path?: string
  version?: string
  last_error?: string
  last_error_code?: string
  last_error_at?: string
}): string {
  const error = status.last_error?.trim()
  if (!error) return ""
  const code = resolveKernelErrorCode(status)
  const lines = [
    status.running === undefined ? "" : `running: ${status.running ? "true" : "false"}`,
    status.config_path?.trim() ? `config: ${status.config_path.trim()}` : "",
    status.version?.trim() ? `version: ${status.version.trim()}` : "",
    code ? `code: ${code}` : "",
    `error: ${error}`,
    status.last_error_at?.trim() ? `at: ${status.last_error_at.trim()}` : "",
  ].filter(Boolean)
  return lines.join("\n")
}
