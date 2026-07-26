/** Dashboard live health summary helpers from connection snapshots. */

import { aggregateConnections, summarizeConnections } from "@/features/observability/connection-stats"
import type { ConnectionWithRates } from "@/features/observability/connection-rate"
import type { Connection, ConnectionEvent, ServiceStatus } from "@/lib/api/types"

export type HealthTone = "ok" | "warn" | "idle" | "offline"

export interface HealthSummary {
  tone: HealthTone
  active: number
  upload: number
  download: number
  uploadRate: number
  downloadRate: number
  rateReady: boolean
  outbounds: number
  topOutbound: string
  topRateOutbound: string
  topRule: string
  tcp: number
  udp: number
  otherNetwork: number
}

export function countNetworks(connections: readonly Connection[]) {
  let tcp = 0
  let udp = 0
  let otherNetwork = 0
  for (const connection of connections) {
    const network = (connection.network ?? "").trim().toLowerCase()
    if (network === "tcp") tcp += 1
    else if (network === "udp") udp += 1
    else otherNetwork += 1
  }
  return { tcp, udp, otherNetwork }
}

export function healthTone(running: boolean | undefined, active: number): HealthTone {
  if (!running) return "offline"
  if (active <= 0) return "idle"
  if (active >= 50) return "warn"
  return "ok"
}

function busiestRateOutbound(connections: ConnectionWithRates[]): string {
  const groups = aggregateConnections(connections, "outbound", Math.max(1, connections.length))
    .filter((group) => group.key !== "—" && group.rateSamples === group.count)
  groups.sort((left, right) => {
    const rateDelta = (right.uploadRate + right.downloadRate) - (left.uploadRate + left.downloadRate)
    return rateDelta || right.count - left.count || left.key.localeCompare(right.key)
  })
  return groups[0]?.key ?? "—"
}

export function buildHealthSummary(
  snapshot: ConnectionEvent | undefined,
  status?: Pick<ServiceStatus, "running">,
): HealthSummary {
  const list: ConnectionWithRates[] = snapshot?.list ?? []
  const active = snapshot?.active_connections ?? list.length
  const totals = summarizeConnections(list)
  const topOutbound = aggregateConnections(list, "outbound", 1)[0]?.key ?? "—"
  const topRateOutbound = busiestRateOutbound(list)
  const topRule = aggregateConnections(list, "rule", 1)[0]?.key ?? "—"
  const networks = countNetworks(list)
  return {
    tone: healthTone(status?.running, active),
    active,
    upload: totals.upload,
    download: totals.download,
    uploadRate: totals.uploadRate,
    downloadRate: totals.downloadRate,
    rateReady: snapshot !== undefined && active === list.length && totals.rateSamples === list.length,
    outbounds: totals.outbounds,
    topOutbound,
    topRateOutbound,
    topRule,
    ...networks,
  }
}
