import { parseConfigError, type ParsedConfigError } from "@/lib/api/config-error"
import { ApiError } from "@/lib/api/client"
import { rolledBackMessage, saveErrorMessage } from "@/lib/api/status"
import type { APIEnvelope } from "@/lib/api/types"
import { classifyKernelErrorMessage } from "@/features/dashboard/kernel-error"

export type ConfigSaveErrorCode =
  | "config_invalid"
  | "config_missing"
  | "restart_failed"
  | "start_failed"
  | "permission"
  | "network"
  | "unknown"

export interface ConfigSaveErrorState {
  message: string
  path?: string
  code?: ConfigSaveErrorCode
  section?: string
  rolledBack?: boolean
}

const HINT_KEYS: Record<ConfigSaveErrorCode, string> = {
  config_invalid: "config.errorHintInvalid",
  config_missing: "config.errorHintMissing",
  restart_failed: "config.errorHintRestartFailed",
  start_failed: "config.errorHintStartFailed",
  permission: "config.errorHintPermission",
  network: "config.errorHintNetwork",
  unknown: "config.errorHintUnknown",
}

export function configSaveErrorHintKey(code?: string): string {
  if (!code) return HINT_KEYS.unknown
  return HINT_KEYS[code as ConfigSaveErrorCode] ?? HINT_KEYS.unknown
}

export function mapConfigApiErrorCode(code?: string): ConfigSaveErrorCode | undefined {
  const value = code?.trim().toLowerCase()
  if (!value) return undefined
  if (value === "config_invalid_runtime" || value === "config_invalid") return "config_invalid"
  if (value === "config_restart_failed" || value === "restart_failed") return "restart_failed"
  if (value === "config_rollback_failed") return "restart_failed"
  if (value === "config_missing") return "config_missing"
  if (value === "permission" || value === "forbidden") return "permission"
  if (value === "unavailable" || value === "bad_gateway" || value === "request_failed") return "network"
  return undefined
}

export function classifyConfigSaveErrorMessage(message?: string, apiCode?: string): ConfigSaveErrorCode {
  const mapped = mapConfigApiErrorCode(apiCode)
  if (mapped) return mapped
  const lower = (message ?? "").toLowerCase()
  if (lower.includes("failed to fetch") || lower.includes("network") || lower.includes("offline")) {
    return "network"
  }
  return classifyKernelErrorMessage(message) as ConfigSaveErrorCode
}

/** Top-level config section from a JSON path like inbounds[0].listen_port */
export function configSectionFromPath(path?: string): string | undefined {
  if (!path) return undefined
  const match = path.trim().match(/^([A-Za-z_][\w-]*)/)
  return match?.[1]
}

export function configSaveErrorFromMessage(
  message: string,
  options: { code?: string; rolledBack?: boolean } = {},
): ConfigSaveErrorState {
  const parsed = parseConfigError(message)
  const code = classifyConfigSaveErrorMessage(parsed.summary || parsed.message, options.code)
  return {
    message: parsed.summary || parsed.message,
    path: parsed.path,
    code,
    section: configSectionFromPath(parsed.path),
    rolledBack: options.rolledBack,
  }
}

export function configSaveErrorFromError(error: unknown, fallback = "request failed"): ConfigSaveErrorState {
  if (error instanceof ApiError) {
    return configSaveErrorFromMessage(saveErrorMessage(error, fallback), { code: error.code })
  }
  if (error instanceof Error) {
    return configSaveErrorFromMessage(saveErrorMessage(error, fallback))
  }
  return configSaveErrorFromMessage(String(error || fallback))
}

export function configSaveErrorFromRollback(
  response: Pick<APIEnvelope<unknown>, "error">,
  fallback: string,
): ConfigSaveErrorState {
  const message = rolledBackMessage(response, fallback)
  return configSaveErrorFromMessage(message, {
    code: response.error?.code || "config_restart_failed",
    rolledBack: true,
  })
}

export function describeConfigSaveError(state: ConfigSaveErrorState | null | undefined): ParsedConfigError | null {
  if (!state?.message) return null
  return {
    message: state.message,
    path: state.path,
    summary: state.message,
  }
}

export function formatConfigSaveErrorTitle(
  t: (key: string, values?: Record<string, string | number>) => string,
  error: ConfigSaveErrorState,
): string {
  if (error.path) return t("config.errorPathTitle")
  if (error.rolledBack) return t("config.rolledBackTitle")
  if (error.code && error.code !== "unknown") {
    return t("config.saveFailedWithCode", { code: error.code })
  }
  return t("config.saveFailedTitle")
}

export function configSaveErrorClipboardText(error: ConfigSaveErrorState | null | undefined): string {
  if (!error?.message?.trim()) return ""
  const code = error.code || classifyConfigSaveErrorMessage(error.message)
  const lines = [
    error.section ? `section: ${error.section}` : "",
    error.path ? `path: ${error.path}` : "",
    code ? `code: ${code}` : "",
    error.rolledBack ? "rolled_back: true" : "",
    `error: ${error.message.trim()}`,
  ].filter(Boolean)
  return lines.join("\n")
}

/** Append ?path= for JSON editor deep-link auto-reveal. */
export function withConfigPathQuery(href: string, path?: string): string {
  const value = path?.trim()
  if (!value) return href
  const joiner = href.includes("?") ? "&" : "?"
  return `${href}${joiner}path=${encodeURIComponent(value)}`
}

/** Always open the raw JSON editor positioned at a config path. */
export function configPathEditorHref(path?: string): string {
  return withConfigPathQuery("/advanced/raw", path)
}

export function configSectionHref(section?: string): string {
  switch (section) {
    case "inbounds":
      return "/proxy/inbounds"
    case "outbounds":
      return "/proxy/outbounds"
    case "route":
      return "/policy/route"
    case "dns":
      return "/policy/dns"
    case "experimental":
      return "/advanced/experimental"
    case "endpoints":
      return "/advanced/endpoints"
    case "certificate":
      return "/advanced/certificate"
    case "services":
      return "/advanced/services"
    case "ntp":
      return "/advanced/ntp"
    default:
      return "/advanced/raw"
  }
}
