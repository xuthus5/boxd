import type { JsonValue } from "@/lib/api/types"

export interface ConfigDiffItem {
  path: string
  kind: "added" | "removed" | "changed"
}

function isObject(value: unknown): value is Record<string, JsonValue> {
  return Boolean(value && typeof value === "object" && !Array.isArray(value))
}

function pathJoin(base: string, key: string) {
  return base ? `${base}.${key}` : key
}

function same(left: unknown, right: unknown) {
  return JSON.stringify(left) === JSON.stringify(right)
}

export function diffConfig(before: unknown, after: unknown, base = "", limit = 40): ConfigDiffItem[] {
  if (same(before, after)) return []
  if (Array.isArray(before) || Array.isArray(after)) {
    if (same(before, after)) return []
    const path = base || "$"
    if (before === undefined) return [{ path, kind: "added" }]
    if (after === undefined) return [{ path, kind: "removed" }]
    return [{ path, kind: "changed" }]
  }
  if (!isObject(before) || !isObject(after)) {
    const path = base || "$"
    if (before === undefined) return [{ path, kind: "added" }]
    if (after === undefined) return [{ path, kind: "removed" }]
    return [{ path, kind: "changed" }]
  }
  const keys = new Set([...Object.keys(before), ...Object.keys(after)])
  const items: ConfigDiffItem[] = []
  for (const key of [...keys].sort()) {
    if (items.length >= limit) break
    const left = before[key]
    const right = after[key]
    if (same(left, right)) continue
    const path = pathJoin(base, key)
    if (left === undefined) {
      items.push({ path, kind: "added" })
      continue
    }
    if (right === undefined) {
      items.push({ path, kind: "removed" })
      continue
    }
    if (isObject(left) && isObject(right)) {
      items.push(...diffConfig(left, right, path, limit - items.length))
      continue
    }
    if (Array.isArray(left) || Array.isArray(right)) {
      items.push({ path, kind: "changed" })
      continue
    }
    items.push({ path, kind: "changed" })
  }
  return items.slice(0, limit)
}

export function summarizeConfigDiff(items: ConfigDiffItem[]) {
  return {
    added: items.filter((item) => item.kind === "added").length,
    removed: items.filter((item) => item.kind === "removed").length,
    changed: items.filter((item) => item.kind === "changed").length,
    total: items.length,
  }
}

export function formatConfigDiffSummary(
  items: ConfigDiffItem[],
  labels: { added: string; removed: string; changed: string; none: string; more: string },
  preview = 6,
) {
  if (!items.length) return labels.none
  const counts = summarizeConfigDiff(items)
  const head = items.slice(0, preview).map((item) => {
    const mark = item.kind === "added" ? "+" : item.kind === "removed" ? "-" : "~"
    return `${mark}${item.path}`
  })
  const extra = items.length > preview ? ` ${labels.more.replace("{{count}}", String(items.length - preview))}` : ""
  return `${labels.changed}: ${counts.changed}, ${labels.added}: ${counts.added}, ${labels.removed}: ${counts.removed}. ${head.join(", ")}${extra}`
}
