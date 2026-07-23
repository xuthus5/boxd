/** 客户端导出/剪贴板失败诊断。 */

export type ExportErrorCode =
  | "clipboard_unavailable"
  | "clipboard_denied"
  | "download_failed"
  | "empty"
  | "unknown"

const HINT_KEYS: Record<ExportErrorCode, string> = {
  clipboard_unavailable: "observability.errorHintClipboardUnavailable",
  clipboard_denied: "observability.errorHintClipboardDenied",
  download_failed: "observability.errorHintDownloadFailed",
  empty: "observability.errorHintExportEmpty",
  unknown: "observability.errorHintExportUnknown",
}

export function exportErrorHintKey(code?: string): string {
  if (!code) return HINT_KEYS.unknown
  return HINT_KEYS[code as ExportErrorCode] ?? HINT_KEYS.unknown
}

export function classifyExportError(error: unknown): ExportErrorCode {
  const message = error instanceof Error ? error.message : String(error || "")
  const lower = message.toLowerCase()
  if (!lower) return "unknown"
  if (lower.includes("clipboard unavailable") || lower.includes("clipboard is not available")) {
    return "clipboard_unavailable"
  }
  if (
    lower.includes("notallowed")
    || lower.includes("not allowed")
    || lower.includes("permission")
    || lower.includes("denied")
    || lower.includes("document is not focused")
  ) {
    return "clipboard_denied"
  }
  if (
    lower.includes("createobjecturl")
    || lower.includes("download")
    || lower.includes("blob")
    || lower.includes("anchor")
  ) {
    return "download_failed"
  }
  if (lower.includes("empty") || lower.includes("no content")) return "empty"
  return "unknown"
}

export function exportErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message.trim()) return error.message.trim()
  return fallback
}

export function formatExportErrorToast(error: unknown, fallback: string): string {
  const message = exportErrorMessage(error, fallback)
  const code = classifyExportError(error)
  if (!code || code === "unknown") return message
  return `${code}: ${message}`
}

export function exportErrorClipboardText(
  error: unknown,
  options: { scope?: string; kind?: string; count?: number; filename?: string } = {},
): string {
  const message = exportErrorMessage(error, "")
  if (!message) return ""
  const code = classifyExportError(error)
  return [
    options.scope ? `scope: ${options.scope}` : "",
    options.kind?.trim() ? `kind: ${options.kind.trim()}` : "",
    Number.isFinite(options.count) ? `count: ${options.count}` : "",
    options.filename?.trim() ? `filename: ${options.filename.trim()}` : "",
    code ? `code: ${code}` : "",
    `error: ${message}`,
  ].filter(Boolean).join("\n")
}
