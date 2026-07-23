import type { Connection } from "@/lib/api/types"
import { ApiError } from "@/lib/api/client"

/** 关闭进行中：单条 / 全量 / 筛选 / 分组。 */
export type ClosingTarget = string | "all" | "filtered" | null

export function isBulkClosing(closingId: ClosingTarget): boolean {
  if (!closingId) return false
  return closingId === "all" || closingId === "filtered" || closingId.startsWith("group:")
}

export function isConnectionRowBusy(closingId: ClosingTarget, id: string): boolean {
  if (!closingId) return false
  if (isBulkClosing(closingId)) return true
  return closingId === id
}

export function isGroupClosing(closingId: ClosingTarget, field: string, key: string): boolean {
  if (!closingId) return false
  if (closingId === "all" || closingId === "filtered") return true
  return closingId === `group:${field}:${key}`
}

export function suppressConnectionIds(
  current: ReadonlySet<string>,
  ids: ReadonlyArray<string | number>,
): Set<string> {
  const next = new Set(current)
  for (const id of ids) next.add(String(id))
  return next
}

/** 仅保留仍出现在 live 快照中的 suppress，避免永久隐藏新连接。 */
export function pruneSuppressedIds(
  suppressed: ReadonlySet<string>,
  liveIds: ReadonlySet<string>,
): Set<string> {
  if (suppressed.size === 0) return new Set()
  let changed = false
  const next = new Set<string>()
  for (const id of suppressed) {
    if (liveIds.has(id)) next.add(id)
    else changed = true
  }
  if (!changed && next.size === suppressed.size) return new Set(suppressed)
  return next
}

export function filterSuppressedConnections(
  connections: readonly Connection[],
  suppressed: ReadonlySet<string>,
): Connection[] {
  if (suppressed.size === 0) return [...connections]
  return connections.filter((item) => !suppressed.has(String(item.id)))
}

export function liveConnectionIdSet(connections: readonly Connection[]): Set<string> {
  return new Set(connections.map((item) => String(item.id)))
}

export function isBenignCloseMiss(error: unknown): boolean {
  return error instanceof ApiError && (error.status === 404 || error.code === "not_found")
}

export function groupClosingKey(field: "outbound" | "rule" | "process", key: string): string {
  return `group:${field}:${key}`
}
