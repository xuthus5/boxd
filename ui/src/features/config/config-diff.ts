import type { JsonValue } from "@/lib/api/types"

export interface ConfigDiffItem {
  path: string
  kind: "added" | "removed" | "changed"
  before?: JsonValue
  after?: JsonValue
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

function asJson(value: unknown): JsonValue | undefined {
  if (value === undefined) return undefined
  return value as JsonValue
}

function leafItem(path: string, before: unknown, after: unknown): ConfigDiffItem {
  if (before === undefined) return { path, kind: "added", after: asJson(after) }
  if (after === undefined) return { path, kind: "removed", before: asJson(before) }
  return { path, kind: "changed", before: asJson(before), after: asJson(after) }
}

export function diffConfig(before: unknown, after: unknown, base = "", limit = 40): ConfigDiffItem[] {
  if (same(before, after)) return []
  if (Array.isArray(before) || Array.isArray(after)) {
    return [leafItem(base || "$", before, after)]
  }
  if (!isObject(before) || !isObject(after)) {
    return [leafItem(base || "$", before, after)]
  }
  const keys = new Set([...Object.keys(before), ...Object.keys(after)])
  const items: ConfigDiffItem[] = []
  for (const key of [...keys].sort()) {
    if (items.length >= limit) break
    const left = before[key]
    const right = after[key]
    if (same(left, right)) continue
    const path = pathJoin(base, key)
    if (left === undefined || right === undefined) {
      items.push(leafItem(path, left, right))
      continue
    }
    if (isObject(left) && isObject(right)) {
      items.push(...diffConfig(left, right, path, limit - items.length))
      continue
    }
    items.push(leafItem(path, left, right))
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

/** Compact single-line preview of a JSON value for diff lists. */
export function previewJsonValue(value: JsonValue | undefined, max = 72): string {
  if (value === undefined) return "—"
  if (value === null) return "null"
  if (typeof value === "string") {
    const text = JSON.stringify(value)
    return text.length > max ? `${text.slice(0, max - 1)}…` : text
  }
  if (typeof value === "number" || typeof value === "boolean") return String(value)
  const text = JSON.stringify(value)
  if (text.length <= max) return text
  if (Array.isArray(value)) return `[…${value.length}]`
  return `{…${Object.keys(value as object).length}}`
}
