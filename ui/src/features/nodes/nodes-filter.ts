/** Node list query / stability filter / sort helpers. */

import { buildLatencyHealth, type LatencyHealth } from "@/features/nodes/latency-health"
import { nodeTestTypes } from "@/features/nodes/node-test-inputs"
import type { LatencyPoint, Outbound } from "@/lib/api/types"

export type NodeStabilityFilter = "" | "stable" | "fair" | "unstable" | "failed" | "unknown"
export type NodeSortKey = "name" | "stability" | "latency"

export type NodeListFilters = {
  query?: string
  stability?: NodeStabilityFilter
  sort?: NodeSortKey
}

export type NodeHistoryMap = Record<string, Record<string, LatencyPoint[]>>

export function matchesNodeQuery(node: Outbound, query: string): boolean {
  const normalized = query.trim().toLowerCase()
  if (!normalized) return true
  const haystack = [node.tag, node.type, node.server ?? "", node.source_name ?? "", String(node.port ?? "")]
    .join(" ")
    .toLowerCase()
  return haystack.includes(normalized)
}

export function pickNodeHistorySeries(history?: Record<string, LatencyPoint[]>): LatencyPoint[] {
  if (!history) return []
  for (const type of nodeTestTypes) {
    const series = history[type]
    if (series?.length) return series
  }
  return Object.values(history)[0] ?? []
}

export function nodeHistoryForTag(history: NodeHistoryMap | undefined, tag: string): LatencyPoint[] {
  return pickNodeHistorySeries(history?.[tag])
}

export function nodeLatencyHealth(node: Outbound, history: NodeHistoryMap | undefined): LatencyHealth {
  return buildLatencyHealth(nodeHistoryForTag(history, node.tag))
}

export function matchesNodeStability(
  health: LatencyHealth,
  stability: NodeStabilityFilter | undefined,
): boolean {
  if (!stability) return true
  switch (stability) {
    case "stable":
      return health.tone === "excellent" || health.tone === "good"
    case "fair":
      return health.tone === "fair"
    case "unstable":
      return health.tone === "poor"
    case "failed":
      return health.tone === "failed"
    case "unknown":
      return health.tone === "unknown"
    default:
      return true
  }
}

function compareByName(left: Outbound, right: Outbound): number {
  return left.tag.localeCompare(right.tag)
}

function compareByStability(
  left: Outbound,
  right: Outbound,
  history: NodeHistoryMap | undefined,
): number {
  const leftHealth = nodeLatencyHealth(left, history)
  const rightHealth = nodeLatencyHealth(right, history)
  const leftRate = leftHealth.rate ?? -1
  const rightRate = rightHealth.rate ?? -1
  if (rightRate !== leftRate) return rightRate - leftRate
  if (rightHealth.count !== leftHealth.count) return rightHealth.count - leftHealth.count
  return compareByName(left, right)
}

function compareByLatency(
  left: Outbound,
  right: Outbound,
  history: NodeHistoryMap | undefined,
): number {
  const leftLatest = nodeLatencyHealth(left, history).latest
  const rightLatest = nodeLatencyHealth(right, history).latest
  if (leftLatest === undefined && rightLatest === undefined) return compareByName(left, right)
  if (leftLatest === undefined) return 1
  if (rightLatest === undefined) return -1
  if (leftLatest !== rightLatest) return leftLatest - rightLatest
  return compareByName(left, right)
}

export function sortNodes(
  nodes: readonly Outbound[],
  sort: NodeSortKey | undefined,
  history: NodeHistoryMap | undefined,
): Outbound[] {
  const items = [...nodes]
  switch (sort) {
    case "stability":
      return items.sort((left, right) => compareByStability(left, right, history))
    case "latency":
      return items.sort((left, right) => compareByLatency(left, right, history))
    case "name":
    default:
      return items.sort(compareByName)
  }
}

export function filterAndSortNodes(
  nodes: readonly Outbound[],
  filters: NodeListFilters,
  history: NodeHistoryMap | undefined,
): Outbound[] {
  const filtered = nodes.filter((node) => {
    if (!matchesNodeQuery(node, filters.query ?? "")) return false
    const health = nodeLatencyHealth(node, history)
    return matchesNodeStability(health, filters.stability)
  })
  return sortNodes(filtered, filters.sort, history)
}

export function nodeFiltersActive(filters: NodeListFilters): boolean {
  return Boolean(filters.query?.trim() || filters.stability)
}

const STABILITY_VALUES = new Set<NodeStabilityFilter>(["stable", "fair", "unstable", "failed", "unknown"])
const SORT_VALUES = new Set<NodeSortKey>(["name", "stability", "latency"])

function readParam(params: { get(name: string): string | null }, key: string): string | undefined {
  const value = params.get(key)?.trim()
  return value ? value : undefined
}

export function parseNodeSearchParams(
  params: URLSearchParams | { get(name: string): string | null },
): NodeListFilters {
  const query = readParam(params, "q")
  const stabilityRaw = readParam(params, "stability") as NodeStabilityFilter | undefined
  const stability = stabilityRaw && STABILITY_VALUES.has(stabilityRaw) ? stabilityRaw : undefined
  const sortRaw = readParam(params, "sort") as NodeSortKey | undefined
  const sort = sortRaw && SORT_VALUES.has(sortRaw) ? sortRaw : undefined
  return { query, stability, sort }
}

export function toNodeSearchParams(filters: NodeListFilters = {}): URLSearchParams {
  const params = new URLSearchParams()
  const query = filters.query?.trim()
  if (query) params.set("q", query)
  if (filters.stability) params.set("stability", filters.stability)
  if (filters.sort && filters.sort !== "name") params.set("sort", filters.sort)
  return params
}

export function buildNodesHref(filters: NodeListFilters = {}): string {
  const qs = toNodeSearchParams(filters).toString()
  return qs ? `/nodes?${qs}` : "/nodes"
}



export type NodeStabilityBucket = Exclude<NodeStabilityFilter, "">

export type NodeStabilitySummary = Record<NodeStabilityBucket, number> & {
  total: number
}

export function stabilityBucketForHealth(health: LatencyHealth): NodeStabilityBucket {
  if (health.tone === "excellent" || health.tone === "good") return "stable"
  if (health.tone === "fair") return "fair"
  if (health.tone === "poor") return "unstable"
  if (health.tone === "failed") return "failed"
  return "unknown"
}

export function summarizeNodeStability(
  nodes: readonly Outbound[],
  history: NodeHistoryMap | undefined,
  query = "",
): NodeStabilitySummary {
  const summary: NodeStabilitySummary = {
    total: 0,
    stable: 0,
    fair: 0,
    unstable: 0,
    failed: 0,
    unknown: 0,
  }
  for (const node of nodes) {
    if (!matchesNodeQuery(node, query)) continue
    summary.total += 1
    summary[stabilityBucketForHealth(nodeLatencyHealth(node, history))] += 1
  }
  return summary
}


export type ProblemNodePreview = {
  tag: string
  type: string
  stability: "unstable" | "failed"
  percent: number
  success: number
  latest?: number
  count: number
}

/** 问题节点预览：全失败优先，其次不稳；默认最多 3 条。 */
export function listProblemNodes(
  nodes: readonly Outbound[] | undefined,
  history: NodeHistoryMap | undefined,
  limit = 3,
): ProblemNodePreview[] {
  if (!Array.isArray(nodes) || nodes.length === 0 || limit <= 0) return []
  const items: ProblemNodePreview[] = []
  for (const node of nodes) {
    const health = nodeLatencyHealth(node, history)
    const bucket = stabilityBucketForHealth(health)
    if (bucket !== "failed" && bucket !== "unstable") continue
    items.push({
      tag: node.tag,
      type: node.type,
      stability: bucket,
      percent: health.percent,
      success: health.success,
      latest: health.latest,
      count: health.count,
    })
  }
  items.sort((left, right) => {
    if (left.stability !== right.stability) {
      return left.stability === "failed" ? -1 : 1
    }
    if (left.percent !== right.percent) return left.percent - right.percent
    if (left.count !== right.count) return right.count - left.count
    return left.tag.localeCompare(right.tag)
  })
  return items.slice(0, Math.floor(limit))
}
