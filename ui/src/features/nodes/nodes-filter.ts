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
