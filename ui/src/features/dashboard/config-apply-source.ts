import { configPathEditorHref, configSectionFromPath, configSectionHref } from "@/features/config/config-save-error"
import { resolveKernelErrorCode } from "@/features/dashboard/kernel-error"
import { extractConfigPath } from "@/lib/api/config-error"

/** Maps backend config apply source codes to i18n keys and ops destinations. */

const sourceKeys: Record<string, string> = {
  update: "sourceUpdate",
  raw: "sourceRaw",
  validate: "sourceValidate",
  validate_raw: "sourceValidateRaw",
  validate_endpoints: "sourceValidateEndpoints",
  validate_ntp: "sourceValidateNTP",
  validate_experimental: "sourceValidateExperimental",
  validate_inbounds: "sourceValidateInbounds",
  validate_outbounds: "sourceValidateOutbounds",
  validate_route: "sourceValidateRoute",
  validate_dns: "sourceValidateDNS",
  dns_defaults: "sourceDNSDefaults",
  route_defaults: "sourceRouteDefaults",
  outbounds_defaults: "sourceOutboundsDefaults",
  inbounds_defaults: "sourceInboundsDefaults",
  experimental_defaults: "sourceExperimentalDefaults",
  rule_sets_defaults: "sourceRuleSetsDefaults",
  restore: "sourceRaw",
}

const sourceHrefs: Record<string, string> = {
  update: "/advanced/raw",
  raw: "/advanced/raw",
  validate: "/advanced/raw",
  validate_raw: "/advanced/raw",
  validate_endpoints: "/advanced/endpoints",
  validate_ntp: "/advanced/ntp",
  validate_experimental: "/advanced/experimental",
  validate_inbounds: "/proxy/inbounds",
  validate_outbounds: "/proxy/outbounds",
  validate_route: "/policy/route",
  validate_dns: "/policy/dns",
  dns_defaults: "/policy/dns",
  route_defaults: "/policy/route",
  outbounds_defaults: "/proxy/outbounds",
  inbounds_defaults: "/proxy/inbounds",
  experimental_defaults: "/advanced/experimental",
  rule_sets_defaults: "/policy/route",
  restore: "/advanced/raw",
}

export function configApplyStatusLabelKey(status: string): string {
  switch (status.trim()) {
    case "rolled_back":
      return "applyStatusRolledBack"
    case "validated":
      return "applyStatusValidated"
    case "validate_failed":
      return "applyStatusValidateFailed"
    default:
      return "applyStatusApplied"
  }
}

export function configApplyStatusVariant(status: string): "destructive" | "secondary" | "outline" {
  switch (status.trim()) {
    case "rolled_back":
    case "validate_failed":
      return "destructive"
    case "validated":
      return "outline"
    default:
      return "secondary"
  }
}

export function configApplyEventFailed(status: string): boolean {
  const value = status.trim()
  return value === "rolled_back" || value === "validate_failed"
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

export function configApplyErrorPath(event: { error?: string }): string | undefined {
  return extractConfigPath(event.error ?? "")
}

export function configApplyErrorSectionHref(event: { error?: string; source?: string }): string {
  const path = configApplyErrorPath(event)
  if (path) return configPathEditorHref(path)
  const section = configSectionFromPath(path)
  if (section) return configSectionHref(section)
  return configApplySourceHref(event.source ?? "")
}

/** Section page without path query (visual editors). */
export function configApplyErrorSectionOnlyHref(event: { error?: string; source?: string }): string {
  const path = configApplyErrorPath(event)
  const section = configSectionFromPath(path)
  if (section) return configSectionHref(section)
  return configApplySourceHref(event.source ?? "")
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
  const path = configApplyErrorPath(event)
  const lines = [
    event.source?.trim() ? `source: ${event.source.trim()}` : "",
    event.status?.trim() ? `status: ${event.status.trim()}` : "",
    event.hash?.trim() ? `hash: ${event.hash.trim()}` : "",
    Number.isFinite(event.size) ? `size: ${event.size}` : "",
    path ? `path: ${path}` : "",
    code ? `code: ${code}` : "",
    event.error?.trim() ? `error: ${event.error.trim()}` : "",
    event.applied_at?.trim() ? `at: ${event.applied_at.trim()}` : "",
  ].filter(Boolean)
  return lines.join("\n")
}
