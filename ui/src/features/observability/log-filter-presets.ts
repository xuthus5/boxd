/** Quick filter presets and deep-link helpers for live log panels. */

export type LogFilterPresetId =
  | "errors"
  | "dns"
  | "connection"
  | "tls"
  | "route"
  | "reject"

export type LogFilterPreset = {
  id: LogFilterPresetId
  labelKey: string
  query: string
  minimum?: "error" | "warn"
}

export type LogThresholdParam = "all" | "debug" | "info" | "warn" | "error"

export type LogSearchFilters = {
  tab?: "kernel" | "application"
  query?: string
  minimum?: LogThresholdParam
  preset?: LogFilterPresetId
}

export const LOG_FILTER_PRESETS: readonly LogFilterPreset[] = [
  { id: "errors", labelKey: "observability.logPresetErrors", query: "error", minimum: "error" },
  { id: "dns", labelKey: "observability.logPresetDNS", query: "dns" },
  { id: "connection", labelKey: "observability.logPresetConnection", query: "inbound outbound connection" },
  { id: "tls", labelKey: "observability.logPresetTLS", query: "tls reality" },
  { id: "route", labelKey: "observability.logPresetRoute", query: "route rule match" },
  { id: "reject", labelKey: "observability.logPresetReject", query: "reject block" },
] as const

const PRESET_IDS = new Set(LOG_FILTER_PRESETS.map((preset) => preset.id))
const THRESHOLDS = new Set<LogThresholdParam>(["all", "debug", "info", "warn", "error"])

export function logPresetById(id: string | null | undefined): LogFilterPreset | undefined {
  if (!id) return undefined
  return LOG_FILTER_PRESETS.find((preset) => preset.id === id)
}

export function matchesLogFilter(level: string, message: string, query: string): boolean {
  const normalized = query.trim().toLowerCase()
  if (!normalized) return true
  const haystack = `${level} ${message}`.toLowerCase()
  // Space-separated tokens: match any token (OR) so multi-keyword presets stay useful.
  const tokens = normalized.split(/\s+/).filter(Boolean)
  if (tokens.length <= 1) return haystack.includes(normalized)
  return tokens.some((token) => haystack.includes(token))
}

export function applyLogPreset(preset: LogFilterPreset | undefined): { filter: string; minimum?: "error" | "warn" } {
  if (!preset) return { filter: "" }
  return { filter: preset.query, minimum: preset.minimum }
}

function readParam(params: { get(name: string): string | null }, key: string): string | undefined {
  const value = params.get(key)?.trim()
  return value ? value : undefined
}

export function parseLogSearchParams(
  params: URLSearchParams | { get(name: string): string | null },
): LogSearchFilters {
  const tabRaw = readParam(params, "tab")
  const tab = tabRaw === "application" || tabRaw === "app" ? "application" : tabRaw === "kernel" ? "kernel" : undefined
  const query = readParam(params, "q")
  const minimumRaw = readParam(params, "minimum")?.toLowerCase() as LogThresholdParam | undefined
  const minimum = minimumRaw && THRESHOLDS.has(minimumRaw) ? minimumRaw : undefined
  const presetRaw = readParam(params, "preset") as LogFilterPresetId | undefined
  const preset = presetRaw && PRESET_IDS.has(presetRaw) ? presetRaw : undefined
  return { tab, query, minimum, preset }
}

export function toLogSearchParams(filters: LogSearchFilters = {}): URLSearchParams {
  const params = new URLSearchParams()
  if (filters.tab && filters.tab !== "kernel") params.set("tab", filters.tab)
  const query = filters.query?.trim()
  if (query) params.set("q", query)
  if (filters.minimum) params.set("minimum", filters.minimum)
  if (filters.preset) params.set("preset", filters.preset)
  return params
}

export function buildLogsHref(filters: LogSearchFilters = {}): string {
  const qs = toLogSearchParams(filters).toString()
  return qs ? `/observability/logs?${qs}` : "/observability/logs"
}

export function resolveLogSeed(filters: LogSearchFilters): {
  filter: string
  minimum?: LogThresholdParam
  preset: LogFilterPresetId | null
} {
  if (filters.preset) {
    const applied = applyLogPreset(logPresetById(filters.preset))
    return {
      filter: filters.query ?? applied.filter,
      minimum: filters.minimum ?? applied.minimum ?? "all",
      preset: filters.preset,
    }
  }
  return {
    filter: filters.query ?? "",
    minimum: filters.minimum,
    preset: null,
  }
}
