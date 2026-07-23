import type { TestResult } from "@/lib/api/types"

export type NodeTestErrorCode =
  | "unavailable"
  | "invalid_input"
  | "timeout"
  | "network"
  | "no_response"
  | "unsupported"
  | "dns_rcode"
  | "empty_response"
  | "unknown"

const HINT_KEYS: Record<string, string> = {
  unavailable: "nodes.errorHintUnavailable",
  invalid_input: "nodes.errorHintInvalidInput",
  timeout: "nodes.errorHintTimeout",
  network: "nodes.errorHintNetwork",
  no_response: "nodes.errorHintNoResponse",
  unsupported: "nodes.errorHintUnsupported",
  dns_rcode: "nodes.errorHintDNSRcode",
  empty_response: "nodes.errorHintEmpty",
  unknown: "nodes.errorHintUnknown",
}

export function nodeTestErrorHintKey(code?: string): string {
  if (!code) return HINT_KEYS.unknown
  return HINT_KEYS[code] ?? HINT_KEYS.unknown
}

export function classifyNodeTestErrorMessage(message?: string): NodeTestErrorCode {
  const lower = (message ?? "").toLowerCase()
  if (!lower) return "unknown"
  if (lower.includes("not available")) return "unavailable"
  if (lower.includes("unsupported") || lower.includes("not probeable")) return "unsupported"
  if (lower.includes("invalid") || lower.includes("required")) return "invalid_input"
  if (lower.includes("no response")) return "no_response"
  if (lower.includes("empty dns") || lower.includes("empty response")) return "empty_response"
  if (lower.includes("dns rcode")) return "dns_rcode"
  if (lower.includes("timeout") || lower.includes("deadline") || lower.includes("i/o timeout")) return "timeout"
  if (
    lower.includes("connection refused")
    || lower.includes("connection reset")
    || lower.includes("no such host")
    || lower.includes("network")
    || lower.includes("ping failed")
    || lower.includes("broken pipe")
  ) return "network"
  return "unknown"
}

export function resolveNodeTestErrorCode(result: {
  success?: boolean
  error?: string
  error_code?: string
}): string | undefined {
  if (result.success) return undefined
  if (!result.error && !result.error_code) return undefined
  return result.error_code || classifyNodeTestErrorMessage(result.error)
}

/** 节点测速失败时的剪贴板诊断文本。 */
export function nodeTestErrorClipboardText(result: Pick<
  TestResult,
  "tag" | "test_type" | "error" | "error_code" | "timestamp" | "success"
>): string {
  if (result.success) return ""
  const code = resolveNodeTestErrorCode(result)
  const lines = [
    result.tag?.trim() ? `tag: ${result.tag.trim()}` : "",
    result.test_type?.trim() ? `test: ${result.test_type.trim()}` : "",
    code ? `code: ${code}` : "",
    result.error?.trim() ? `error: ${result.error.trim()}` : "",
    result.timestamp?.trim() ? `at: ${result.timestamp.trim()}` : "",
  ].filter(Boolean)
  return lines.join("\n")
}

export function nodeTestErrorLabel(result: Pick<TestResult, "error" | "error_code">, fallback: string): string {
  const message = result.error?.trim()
  if (message) return message
  const code = result.error_code?.trim()
  if (code) return code
  return fallback
}

export function formatNodeTestFailureSample(result: Pick<TestResult, "error" | "error_code">): string {
  const code = resolveNodeTestErrorCode({ ...result, success: false })
  const error = result.error?.trim() || "failed"
  if (!code || code === "unknown" || code === error) return error
  return `${code}: ${error}`
}
