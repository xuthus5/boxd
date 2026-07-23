import type { DNSProbeResult } from "@/lib/api/types"

export type DNSProbeErrorCode =
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
  unavailable: "policy.dns.errorHintUnavailable",
  invalid_input: "policy.dns.errorHintInvalidInput",
  timeout: "policy.dns.errorHintTimeout",
  network: "policy.dns.errorHintNetwork",
  no_response: "policy.dns.errorHintNoResponse",
  unsupported: "policy.dns.errorHintUnsupported",
  dns_rcode: "policy.dns.errorHintDNSRcode",
  empty_response: "policy.dns.errorHintEmpty",
  unknown: "policy.dns.errorHintUnknown",
}

export function dnsProbeErrorHintKey(code?: string): string {
  if (!code) return HINT_KEYS.unknown
  return HINT_KEYS[code] ?? HINT_KEYS.unknown
}

export function classifyDNSProbeErrorMessage(message?: string): DNSProbeErrorCode {
  const lower = (message ?? "").toLowerCase()
  if (!lower) return "unknown"
  if (lower.includes("not available")) return "unavailable"
  if (lower.includes("unsupported") || lower.includes("not probeable")) return "unsupported"
  if (lower.includes("invalid") || lower.includes("required") || lower.includes("empty address")) return "invalid_input"
  if (lower.includes("no response")) return "no_response"
  if (lower.includes("empty dns") || lower.includes("empty response")) return "empty_response"
  if (lower.includes("dns rcode")) return "dns_rcode"
  if (lower.includes("timeout") || lower.includes("deadline") || lower.includes("i/o timeout")) return "timeout"
  if (
    lower.includes("connection refused")
    || lower.includes("connection reset")
    || lower.includes("no such host")
    || lower.includes("network")
    || lower.includes("broken pipe")
  ) return "network"
  return "unknown"
}

export function resolveDNSProbeErrorCode(result: {
  success?: boolean
  error?: string
  error_code?: string
}): string | undefined {
  if (result.success) return undefined
  if (!result.error && !result.error_code) return undefined
  return result.error_code || classifyDNSProbeErrorMessage(result.error)
}

export function dnsProbeErrorClipboardText(result: Pick<
  DNSProbeResult,
  "tag" | "type" | "error" | "error_code" | "domain" | "success"
>): string {
  if (result.success) return ""
  const code = resolveDNSProbeErrorCode(result)
  const lines = [
    result.tag?.trim() ? `tag: ${result.tag.trim()}` : "",
    result.type?.trim() ? `type: ${result.type.trim()}` : "",
    result.domain?.trim() ? `domain: ${result.domain.trim()}` : "",
    code ? `code: ${code}` : "",
    result.error?.trim() ? `error: ${result.error.trim()}` : "",
  ].filter(Boolean)
  return lines.join("\n")
}

export function formatDNSProbeFailureSample(result: Pick<DNSProbeResult, "error" | "error_code">): string {
  const code = resolveDNSProbeErrorCode({ ...result, success: false })
  const error = result.error?.trim() || "failed"
  if (!code || code === "unknown" || code === error) return error
  return `${code}: ${error}`
}

export function classifyDNSProbeRequestError(error: unknown): DNSProbeErrorCode {
  if (error && typeof error === "object" && "code" in error) {
    const code = String((error as { code?: string }).code || "").toLowerCase()
    if (code === "unavailable") return "unavailable"
    if (code === "invalid_request" || code === "invalid_input") return "invalid_input"
    if (code === "timeout") return "timeout"
    if (code === "unsupported") return "unsupported"
  }
  const message = error instanceof Error ? error.message : String(error || "")
  return classifyDNSProbeErrorMessage(message)
}

export function dnsProbeRequestErrorClipboardText(error: unknown): string {
  const message = error instanceof Error ? error.message.trim() : String(error || "").trim()
  if (!message) return ""
  const code = classifyDNSProbeRequestError(error)
  return [`code: ${code}`, `error: ${message}`].join("\n")
}

export function formatDNSProbeRequestErrorToast(
  error: unknown,
  _t: (key: string, values?: Record<string, string | number>) => string,
  fallback: string,
): string {
  const message = error instanceof Error && error.message.trim() ? error.message.trim() : fallback
  const code = classifyDNSProbeRequestError(error)
  if (!code || code === "unknown") return message
  return `${code}: ${message}`
}

