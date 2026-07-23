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
  level?: string
  preset?: LogFilterPresetId
}

export type LogLevelBucket = {
  level: string
  count: number
}

export type LogLevelSummary = {
  total: number
  buckets: LogLevelBucket[]
}

const LEVEL_ORDER = ["error", "warn", "info", "debug"] as const

export function normalizeLogLevel(level: string | undefined): string {
  const value = level?.trim().toLowerCase()
  return value || "unknown"
}

export function matchesLogLevel(level: string | undefined, selected: string | undefined): boolean {
  if (!selected) return true
  return normalizeLogLevel(level) === selected.trim().toLowerCase()
}

export function summarizeLogLevels(
  items: readonly { level?: string; message?: string }[],
  query = "",
): LogLevelSummary {
  const counts = new Map<string, number>()
  let total = 0
  for (const item of items) {
    if (!matchesLogFilter(item.level ?? "", item.message ?? "", query)) continue
    total += 1
    const level = normalizeLogLevel(item.level)
    counts.set(level, (counts.get(level) ?? 0) + 1)
  }
  const buckets = Array.from(counts.entries())
    .map(([level, count]) => ({ level, count }))
    .sort((left, right) => {
      const leftRank = LEVEL_ORDER.indexOf(left.level as (typeof LEVEL_ORDER)[number])
      const rightRank = LEVEL_ORDER.indexOf(right.level as (typeof LEVEL_ORDER)[number])
      const leftOrder = leftRank === -1 ? LEVEL_ORDER.length : leftRank
      const rightOrder = rightRank === -1 ? LEVEL_ORDER.length : rightRank
      if (leftOrder !== rightOrder) return leftOrder - rightOrder
      if (right.count !== left.count) return right.count - left.count
      return left.level.localeCompare(right.level)
    })
  return { total, buckets }
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
  const level = readParam(params, "level")?.toLowerCase()
  const presetRaw = readParam(params, "preset") as LogFilterPresetId | undefined
  const preset = presetRaw && PRESET_IDS.has(presetRaw) ? presetRaw : undefined
  return { tab, query, minimum, level, preset }
}

export function toLogSearchParams(filters: LogSearchFilters = {}): URLSearchParams {
  const params = new URLSearchParams()
  if (filters.tab && filters.tab !== "kernel") params.set("tab", filters.tab)
  const query = filters.query?.trim()
  if (query) params.set("q", query)
  if (filters.minimum) params.set("minimum", filters.minimum)
  const level = filters.level?.trim().toLowerCase()
  if (level) params.set("level", level)
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
  level?: string
  preset: LogFilterPresetId | null
} {
  if (filters.preset) {
    const applied = applyLogPreset(logPresetById(filters.preset))
    return {
      filter: filters.query ?? applied.filter,
      minimum: filters.minimum ?? applied.minimum ?? "all",
      level: filters.level,
      preset: filters.preset,
    }
  }
  return {
    filter: filters.query ?? "",
    minimum: filters.minimum,
    level: filters.level,
    preset: null,
  }
}
