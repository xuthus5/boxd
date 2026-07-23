import { formatLatency } from "@/features/nodes/node-format"
import { api } from "@/lib/api/endpoints"
import type { OutboundGroup } from "@/lib/api/types"

export type DelayMap = Record<string, number | "error">

const preferredTags = ["proxy", "select", "GLOBAL"]

export function pickPrimaryGroup(groups: OutboundGroup[]) {
  const selectors = groups.filter((group) => group.type === "selector" && group.all.length > 0)
  if (!selectors.length) return null
  for (const tag of preferredTags) {
    const found = selectors.find((group) => group.tag === tag)
    if (found) return found
  }
  return selectors[0]
}

export function formatDelayValue(value: number | "error" | undefined, failedLabel: string) {
  if (value === "error") return failedLabel
  if (typeof value === "number") return formatLatency(value)
  return "—"
}

export function sortDelayEntries(delays: DelayMap) {
  return Object.entries(delays).sort((left, right) => {
    const leftOk = typeof left[1] === "number"
    const rightOk = typeof right[1] === "number"
    if (leftOk !== rightOk) return leftOk ? -1 : 1
    if (leftOk && rightOk && left[1] !== right[1]) return Number(left[1]) - Number(right[1])
    return left[0].localeCompare(right[0])
  })
}

export function summarizeDelays(delays: DelayMap) {
  let ok = 0
  let failed = 0
  for (const value of Object.values(delays)) {
    if (typeof value === "number") ok += 1
    else failed += 1
  }
  return { total: ok + failed, ok, failed }
}

async function probeMemberDelays(members: readonly string[]): Promise<DelayMap> {
  const entries = await Promise.all(members.map(async (tag) => {
    try {
      const result = await api.nodes.delay(tag) as { delay?: number }
      return [tag, typeof result.delay === "number" ? result.delay : "error"] as const
    } catch {
      return [tag, "error"] as const
    }
  }))
  return Object.fromEntries(entries)
}

export async function measureGroupDelays(groupTag: string, members: readonly string[]): Promise<DelayMap> {
  try {
    const urlTest = await api.nodes.urlTest(groupTag)
    const next: DelayMap = {}
    for (const tag of members) {
      const value = urlTest[tag]
      next[tag] = typeof value === "number" ? value : "error"
    }
    return next
  } catch {
    return probeMemberDelays(members)
  }
}
