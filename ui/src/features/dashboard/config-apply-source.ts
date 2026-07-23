import { resolveKernelErrorCode } from "@/features/dashboard/kernel-error"

/** Maps backend config apply source codes to i18n keys and ops destinations. */

const sourceKeys: Record<string, string> = {
  update: "sourceUpdate",
  raw: "sourceRaw",
  dns_defaults: "sourceDNSDefaults",
  route_defaults: "sourceRouteDefaults",
  outbounds_defaults: "sourceOutboundsDefaults",
  inbounds_defaults: "sourceInboundsDefaults",
  experimental_defaults: "sourceExperimentalDefaults",
  rule_sets_defaults: "sourceRuleSetsDefaults",
}

const sourceHrefs: Record<string, string> = {
  update: "/advanced/raw",
  raw: "/advanced/raw",
  dns_defaults: "/policy/dns",
  route_defaults: "/policy/route",
  outbounds_defaults: "/proxy/outbounds",
  inbounds_defaults: "/proxy/inbounds",
  experimental_defaults: "/advanced/experimental",
  rule_sets_defaults: "/policy/route",
}

export function configApplySourceKey(source: string): string {
  return sourceKeys[source] ?? "sourceUnknown"
}

export function configApplySourceHref(source: string): string {
  return sourceHrefs[source] ?? "/advanced/raw"
}

export function shortConfigHash(hash: string, length = 8): string {
  const value = hash.trim()
  if (!value) return "—"
  return value.slice(0, length)
}

export function configApplyErrorClipboardText(event: {
  source?: string
  status?: string
  hash?: string
  size?: number
  error?: string
  error_code?: string
  applied_at?: string
}): string {
  const code = resolveKernelErrorCode(event)
  const lines = [
    event.source?.trim() ? `source: ${event.source.trim()}` : "",
    event.status?.trim() ? `status: ${event.status.trim()}` : "",
    event.hash?.trim() ? `hash: ${event.hash.trim()}` : "",
    Number.isFinite(event.size) ? `size: ${event.size}` : "",
    code ? `code: ${code}` : "",
    event.error?.trim() ? `error: ${event.error.trim()}` : "",
    event.applied_at?.trim() ? `at: ${event.applied_at.trim()}` : "",
  ].filter(Boolean)
  return lines.join("\n")
}

