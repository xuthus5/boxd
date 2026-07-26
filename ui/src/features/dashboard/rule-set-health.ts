import type { LogEvent, RuleSetAutoUpdate, RuleSetStatusItem } from "@/lib/api/types"

export type RuleSetArtifactState = "ready" | "missing" | "stale" | "unmanaged"
export type RuleSetHealthTone = "healthy" | "warning" | "error" | "empty"

export interface RuleSetHealthItem {
  index: number
  tag: string
  state: RuleSetArtifactState
  item: RuleSetStatusItem
}

export interface AutoUpdateRun {
  updated: number
  failed: number
  skipped: number
  error?: string
  timestamp?: string
}

export interface RuleSetHealthSummary {
  tone: RuleSetHealthTone
  total: number
  updatable: number
  available: number
  missing: number
  stale: number
  unmanaged: number
  latestUpdatedAt?: string
  items: RuleSetHealthItem[]
  latestAutoUpdate?: AutoUpdateRun
}

const DURATION_PART = /(\d+(?:\.\d+)?)(ms|s|m|h|d)/gy
const DURATION_MULTIPLIERS: Record<string, number> = {
  ms: 1,
  s: 1000,
  m: 60 * 1000,
  h: 60 * 60 * 1000,
  d: 24 * 60 * 60 * 1000,
}
const STALE_GRACE_FACTOR = 1.25

export function parseRuleSetDuration(value?: string): number | undefined {
  const input = value?.trim()
  if (!input) return undefined
  DURATION_PART.lastIndex = 0
  let cursor = 0
  let total = 0
  let match: RegExpExecArray | null
  while ((match = DURATION_PART.exec(input)) !== null) {
    if (match.index !== cursor) return undefined
    const amount = Number(match[1])
    const multiplier = DURATION_MULTIPLIERS[match[2]]
    if (!Number.isFinite(amount) || amount <= 0 || multiplier === undefined) return undefined
    total += amount * multiplier
    cursor = DURATION_PART.lastIndex
  }
  if (cursor !== input.length || !Number.isFinite(total) || total <= 0) return undefined
  return total
}

function metric(message: string, name: string): number | undefined {
  const match = new RegExp(`${name}=(\\d+)\\b`).exec(message)
  if (!match) return undefined
  const value = Number(match[1])
  return Number.isSafeInteger(value) ? value : undefined
}

export function parseAutoUpdateLog(entry: LogEvent): AutoUpdateRun | undefined {
  if (!entry || typeof entry.message !== "string") return undefined
  if (entry.message.includes("ruleset auto update failed")) {
    const error = /\berr=(.+)$/.exec(entry.message)?.[1]?.trim() || "failed"
    return { updated: 0, failed: 1, skipped: 0, error, timestamp: entry.timestamp }
  }
  if (!entry.message.includes("ruleset auto update finished")) return undefined
  const updated = metric(entry.message, "updated")
  const failed = metric(entry.message, "failed")
  const skipped = metric(entry.message, "skipped")
  if (updated === undefined || failed === undefined || skipped === undefined) return undefined
  return { updated, failed, skipped, timestamp: entry.timestamp }
}

function hasArtifact(item: RuleSetStatusItem): boolean {
  return Boolean(item.last_updated) || (item.file_size ?? 0) > 0
}

function staleAfter(item: RuleSetStatusItem, autoUpdate: RuleSetAutoUpdate): number | undefined {
  const interval = item.update_interval || (item.type === "local" && item.builtin && autoUpdate.enabled ? autoUpdate.interval : undefined)
  const duration = parseRuleSetDuration(interval)
  return duration === undefined ? undefined : duration * STALE_GRACE_FACTOR
}

function isStale(item: RuleSetStatusItem, autoUpdate: RuleSetAutoUpdate, now: Date): boolean {
  if (!item.last_updated) return false
  const updatedAt = new Date(item.last_updated)
  const threshold = staleAfter(item, autoUpdate)
  if (Number.isNaN(updatedAt.getTime()) || threshold === undefined) return false
  return now.getTime() - updatedAt.getTime() > threshold
}

function classifyItem(item: RuleSetStatusItem, autoUpdate: RuleSetAutoUpdate, now: Date): RuleSetArtifactState {
  if (!item.updatable) return "unmanaged"
  if (!hasArtifact(item)) return "missing"
  return isStale(item, autoUpdate, now) ? "stale" : "ready"
}

function latestAutoUpdate(logs: LogEvent[]): AutoUpdateRun | undefined {
  for (let index = logs.length - 1; index >= 0; index -= 1) {
    const parsed = parseAutoUpdateLog(logs[index])
    if (parsed) return parsed
  }
  return undefined
}

function latestArtifactUpdatedAt(items: RuleSetStatusItem[]): string | undefined {
  let latestValue: string | undefined
  let latestTime = Number.NEGATIVE_INFINITY
  for (const item of items) {
    if (!item.last_updated) continue
    const timestamp = Date.parse(item.last_updated)
    if (!Number.isFinite(timestamp) || timestamp <= latestTime) continue
    latestTime = timestamp
    latestValue = item.last_updated
  }
  return latestValue
}

function toneFor(
  total: number,
  missing: number,
  stale: number,
  latest: AutoUpdateRun | undefined,
): RuleSetHealthTone {
  if (total === 0) return "empty"
  if ((latest?.failed ?? 0) > 0) return "error"
  if (missing > 0 || stale > 0) return "warning"
  return "healthy"
}

export function buildRuleSetHealth(
  items: RuleSetStatusItem[],
  autoUpdate: RuleSetAutoUpdate,
  logs: LogEvent[],
  now = new Date(),
): RuleSetHealthSummary {
  const healthItems = items.map((item, index) => ({
    index,
    tag: item.tag || `#${index + 1}`,
    state: classifyItem(item, autoUpdate, now),
    item,
  }))
  const missing = healthItems.filter((entry) => entry.state === "missing").length
  const stale = healthItems.filter((entry) => entry.state === "stale").length
  const unmanaged = healthItems.filter((entry) => entry.state === "unmanaged").length
  const latest = latestAutoUpdate(logs)
  return {
    tone: toneFor(items.length, missing, stale, latest),
    total: items.length,
    updatable: items.filter((item) => item.updatable).length,
    available: items.filter(hasArtifact).length,
    missing,
    stale,
    unmanaged,
    latestUpdatedAt: latestArtifactUpdatedAt(items),
    items: healthItems,
    latestAutoUpdate: latest,
  }
}

export function ruleSetHealthHref(index?: number): string {
  if (index === undefined) return "/policy/route"
  return `/policy/route?path=${encodeURIComponent(`route.rule_set[${index}]`)}`
}
