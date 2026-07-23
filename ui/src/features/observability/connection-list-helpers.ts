/** Helpers for connection list rendering and deep-links. */

import {
  type ConnectionColumnId,
} from "@/features/observability/connection-columns"
import { formatBytes } from "@/features/dashboard/format"
import { buildNodesHref } from "@/features/nodes/nodes-filter"
import { connectionTargetLogQuery } from "@/features/observability/connection-facets"
import { buildLogsHref } from "@/features/observability/log-filter-presets"
import { buildRouteHref } from "@/features/policy/route-rule-filter"
import type { Connection } from "@/lib/api/types"

export function targetLogsHref(target?: string) {
  const query = connectionTargetLogQuery(target)
  return query ? buildLogsHref({ query }) : ""
}

export function nodeHref(outbound?: string) {
  const tag = outbound?.trim()
  if (!tag || tag === "—") return ""
  return buildNodesHref({ query: tag })
}

export function ruleRouteHref(rule?: string) {
  const value = rule?.trim()
  if (!value || value === "—") return ""
  return buildRouteHref({ query: value })
}

export function formatDuration(start: string) {
  const startedAt = new Date(start).getTime()
  if (!Number.isFinite(startedAt)) return "—"
  return `${Math.floor(Math.max(0, Date.now() - startedAt) / 1000)}s`
}

export function cellValue(connection: Connection, id: ConnectionColumnId, duration: string): string {
  switch (id) {
    case "target":
      return connection.target || "—"
    case "source":
      return connection.source || "—"
    case "network":
      return connection.network || "—"
    case "inbound":
      return connection.inbound || "—"
    case "outbound":
      return connection.outbound || "—"
    case "rule":
      return connection.rule || "—"
    case "protocol":
      return connection.protocol || "—"
    case "process":
      return connection.process || "—"
    case "upload":
      return formatBytes(connection.upload)
    case "download":
      return formatBytes(connection.download)
    case "duration":
      return duration
    default:
      return "—"
  }
}

export function titleFor(connection: Connection, id: ConnectionColumnId): string | undefined {
  if (id === "target") return connection.target || undefined
  if (id === "source") return connection.source || undefined
  if (id === "inbound") return connection.inbound || undefined
  if (id === "rule") return connection.rule || undefined
  if (id === "process") return connection.process || undefined
  return undefined
}
