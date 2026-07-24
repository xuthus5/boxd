import { ApiError } from "@/lib/api/client"

export type NodeRequestErrorCode =
  | "unavailable"
  | "invalid_input"
  | "not_found"
  | "conflict"
  | "update_failed"
  | "network"
  | "timeout"
  | "unsupported"
  | "unknown"

const HINT_KEYS: Record<NodeRequestErrorCode, string> = {
  unavailable: "nodes.errorHintRequestUnavailable",
  invalid_input: "nodes.errorHintRequestInvalid",
  not_found: "nodes.errorHintRequestNotFound",
  conflict: "nodes.errorHintRequestConflict",
  update_failed: "nodes.errorHintRequestUpdateFailed",
  network: "nodes.errorHintRequestNetwork",
  timeout: "nodes.errorHintRequestTimeout",
  unsupported: "nodes.errorHintRequestUnsupported",
  unknown: "nodes.errorHintRequestUnknown",
}

export function nodeRequestErrorHintKey(code?: string): string {
  if (!code) return HINT_KEYS.unknown
  return HINT_KEYS[code as NodeRequestErrorCode] ?? HINT_KEYS.unknown
}

export function classifyNodeRequestError(error: unknown): NodeRequestErrorCode {
  if (error instanceof ApiError) {
    const code = error.code?.toLowerCase() || ""
    if (code === "unavailable") return "unavailable"
    if (code === "invalid_request") return "invalid_input"
    if (code === "node_not_found" || code === "not_found" || code === "runtime_group_not_found") return "not_found"
    if (code === "conflict" || code === "node_tag_conflict") return "conflict"
    if (code === "node_update_failed") return "update_failed"
    if (code === "runtime_not_selectable") return "unsupported"
    if (code === "timeout") return "timeout"
    if (code === "bad_gateway" || code === "request_failed") return "network"
  }
  const message = error instanceof Error ? error.message : String(error || "")
  const lower = message.toLowerCase()
  if (!lower) return "unknown"
  if (lower.includes("not available") || lower.includes("not running")) return "unavailable"
  if (lower.includes("not found")) return "not_found"
  if (lower.includes("invalid") || lower.includes("required") || lower.includes("parse")) return "invalid_input"
  if (lower.includes("timeout") || lower.includes("deadline")) return "timeout"
  if (lower.includes("unsupported") || lower.includes("not selectable")) return "unsupported"
  if (
    lower.includes("failed to fetch")
    || lower.includes("network")
    || lower.includes("connection refused")
    || lower.includes("offline")
  ) return "network"
  if (lower.includes("failed to") || lower.includes("update")) return "update_failed"
  return "unknown"
}

export function nodeRequestErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message.trim()) return error.message.trim()
  return fallback
}

export function formatNodeRequestErrorToast(
  error: unknown,
  fallback: string,
): string {
  const message = nodeRequestErrorMessage(error, fallback)
  const code = classifyNodeRequestError(error)
  if (!code || code === "unknown") return message
  return `${code}: ${message}`
}

export function nodeRequestErrorClipboardText(
  error: unknown,
  options: { scope?: string; tag?: string; group?: string } = {},
): string {
  const message = nodeRequestErrorMessage(error, "")
  if (!message) return ""
  const code = classifyNodeRequestError(error)
  return [
    options.scope ? `scope: ${options.scope}` : "",
    options.group?.trim() ? `group: ${options.group.trim()}` : "",
    options.tag?.trim() ? `tag: ${options.tag.trim()}` : "",
    code ? `code: ${code}` : "",
    `error: ${message}`,
  ].filter(Boolean).join("\n")
}
