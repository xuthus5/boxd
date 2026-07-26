const TRAFFIC_WINDOW_MS = 60_000
export const TRAFFIC_CHART_POINT_LIMIT = 60
export const TRAFFIC_CHART_SOURCE_LIMIT = TRAFFIC_CHART_POINT_LIMIT + 1
const TRAFFIC_DEFAULT_MAX = 1
const TRAFFIC_DOMAIN_HEADROOM = 1.1
const TRAFFIC_CHART_LEFT_MARGIN = 20
const TRAFFIC_Y_AXIS_WIDTH = 72
const TRAFFIC_TIME_FORMATTER = new Intl.DateTimeFormat(undefined, {
  hour: "2-digit",
  minute: "2-digit",
  second: "2-digit",
  hour12: false,
})

export const TRAFFIC_UPDATE_DURATION_MS = 1000
export const TRAFFIC_CHART_MARGIN = { top: 4, right: 8, bottom: 0, left: TRAFFIC_CHART_LEFT_MARGIN }
export const TRAFFIC_AXIS_WIDTH = TRAFFIC_Y_AXIS_WIDTH

export type TrafficChartPoint = { timestamp: string }
export type TrafficDomain = readonly [number, number]

function timestampToMilliseconds(value: unknown) {
  if (typeof value !== "string") return 0
  const timestamp = Date.parse(value)
  return Number.isFinite(timestamp) ? timestamp : 0
}

export function getTrafficTimestampValue(point: TrafficChartPoint) {
  return timestampToMilliseconds(point.timestamp)
}

export function formatTrafficTimestamp(value: unknown) {
  const timestamp = typeof value === "number" ? value : timestampToMilliseconds(value)
  return timestamp > 0 ? TRAFFIC_TIME_FORMATTER.format(timestamp) : ""
}

export function getTrafficTimeDomain(data: readonly TrafficChartPoint[]): TrafficDomain {
  const latest = data.reduce((current, point) => Math.max(current, getTrafficTimestampValue(point)), 0)
  if (latest <= 0) return [0, TRAFFIC_WINDOW_MS]
  return [latest - TRAFFIC_WINDOW_MS, latest]
}

export function getTrafficPointValue(point: TrafficChartPoint, dataKey: string) {
  const value = (point as unknown as Record<string, unknown>)[dataKey]
  return typeof value === "number" && Number.isFinite(value) ? value : undefined
}

function niceTrafficMaximum(value: number) {
  if (value <= 0) return TRAFFIC_DEFAULT_MAX
  const maximum = 2 ** Math.ceil(Math.log2(value * TRAFFIC_DOMAIN_HEADROOM))
  return Number.isFinite(maximum) ? maximum : value
}

export function getTrafficValueDomain(data: readonly TrafficChartPoint[], dataKeys: readonly string[]): TrafficDomain {
  let maximum = 0
  for (const point of data) {
    for (const dataKey of dataKeys) {
      maximum = Math.max(maximum, getTrafficPointValue(point, dataKey) ?? 0)
    }
  }
  return [0, niceTrafficMaximum(maximum)]
}
