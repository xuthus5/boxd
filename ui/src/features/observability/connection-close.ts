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

export type CloseScope = "one" | "all" | "filtered" | "group"

export function formatClosedScopeMessage(
  scope: CloseScope,
  t: (key: string, values?: Record<string, string | number>) => string,
  options: { count?: number; target?: string; group?: string } = {},
): string {
  if (scope === "one") {
    const target = options.target?.trim()
    if (target) return t("observability.closedOneTarget", { target })
    return t("observability.closedOne")
  }
  if (scope === "all") {
    return t("observability.closedAllDone", { count: options.count ?? 0 })
  }
  if (scope === "filtered") {
    return t("observability.closedFilteredDone", { count: options.count ?? 0 })
  }
  return t("observability.closeGroupDoneCount", {
    group: options.group ?? "—",
    count: options.count ?? 0,
  })
}

export type CloseErrorCode =
  | "unavailable"
  | "not_found"
  | "invalid_request"
  | "unknown"

export function classifyCloseError(error: unknown): CloseErrorCode {
  if (error instanceof ApiError) {
    const code = error.code?.toLowerCase() || ""
    if (code === "unavailable") return "unavailable"
    if (code === "not_found") return "not_found"
    if (code === "invalid_request") return "invalid_request"
    const lower = error.message.toLowerCase()
    if (lower.includes("not available") || lower.includes("unavailable")) return "unavailable"
    if (lower.includes("not found")) return "not_found"
    if (lower.includes("invalid") || lower.includes("too many") || lower.includes("specify only one")) {
      return "invalid_request"
    }
  }
  if (error instanceof Error) {
    const lower = error.message.toLowerCase()
    if (lower.includes("not available") || lower.includes("unavailable")) return "unavailable"
    if (lower.includes("not found")) return "not_found"
    if (lower.includes("invalid") || lower.includes("too many")) return "invalid_request"
  }
  return "unknown"
}

export function closeErrorHintKey(code?: string): string {
  switch (code) {
    case "unavailable":
      return "observability.errorHintCloseUnavailable"
    case "not_found":
      return "observability.errorHintCloseNotFound"
    case "invalid_request":
      return "observability.errorHintCloseInvalid"
    default:
      return "observability.errorHintCloseUnknown"
  }
}

export function closeErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message.trim()) return error.message.trim()
  return fallback
}

export function formatCloseErrorToast(
  error: unknown,
  t: (key: string, values?: Record<string, string | number>) => string,
  options: { scope: CloseScope; target?: string; group?: string } ,
): string {
  const code = classifyCloseError(error)
  const detail = closeErrorMessage(error, t("observability.closeFailed"))
  const scopeLabel =
    options.scope === "one"
      ? (options.target?.trim() || t("observability.closedOne"))
      : options.scope === "group"
        ? (options.group?.trim() || "—")
        : options.scope
  const head = t("observability.closeFailedScope", {
    scope: scopeLabel,
    code: code === "unknown" ? "—" : code,
  })
  return `${head}: ${detail}`
}

export function closeErrorClipboardText(error: unknown, options: {
  scope: CloseScope
  target?: string
  group?: string
  count?: number
}): string {
  const code = classifyCloseError(error)
  const detail = closeErrorMessage(error, "close failed")
  const lines = [
    `scope: ${options.scope}`,
    options.target?.trim() ? `target: ${options.target.trim()}` : "",
    options.group?.trim() ? `group: ${options.group.trim()}` : "",
    options.count !== undefined ? `count: ${options.count}` : "",
    code ? `code: ${code}` : "",
    `error: ${detail}`,
  ].filter(Boolean)
  return lines.join("\n")
}
