import { formatBytes } from "@/features/dashboard/format"
import type { Connection } from "@/lib/api/types"

export type ConnectionWithRates = Connection & {
  uploadRate?: number
  downloadRate?: number
}

function sameConnection(current: Connection, previous: Connection): boolean {
  return current.id === previous.id && current.start === previous.start
}

function calculateRate(current: number, previous: number, elapsedMs: number): number | undefined {
  if (!Number.isFinite(current) || !Number.isFinite(previous) || !Number.isFinite(elapsedMs) || elapsedMs <= 0) {
    return undefined
  }
  const delta = current - previous
  return delta < 0 ? undefined : delta * 1000 / elapsedMs
}

export function calculateConnectionRates(
  current: readonly Connection[],
  previous: readonly Connection[],
  elapsedMs: number,
): ConnectionWithRates[] {
  const previousByID = new Map(previous.map((connection) => [connection.id, connection]))
  return current.map((connection) => {
    const baseline = previousByID.get(connection.id)
    if (!baseline || !sameConnection(connection, baseline)) return { ...connection }
    const uploadRate = calculateRate(connection.upload, baseline.upload, elapsedMs)
    const downloadRate = calculateRate(connection.download, baseline.download, elapsedMs)
    if (uploadRate === undefined || downloadRate === undefined) return { ...connection }
    return { ...connection, uploadRate, downloadRate }
  })
}

export function connectionRateTotal(connection: ConnectionWithRates): number | undefined {
  if (connection.uploadRate === undefined || connection.downloadRate === undefined) return undefined
  return connection.uploadRate + connection.downloadRate
}

export function formatConnectionRate(value: number | undefined): string {
  if (value === undefined) return "—"
  const displayValue = value < 1024 ? Math.round(value) : value
  return `${formatBytes(displayValue)}/s`
}

export function formatConnectionRatePair(uploadRate?: number, downloadRate?: number): string {
  if (uploadRate === undefined || downloadRate === undefined) return "—"
  return `↑ ${formatConnectionRate(uploadRate)} · ↓ ${formatConnectionRate(downloadRate)}`
}
