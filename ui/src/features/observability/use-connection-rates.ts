import { useEffect, useMemo, useRef, useState } from "react"

import {
  calculateConnectionRates,
  type ConnectionWithRates,
} from "@/features/observability/connection-rate"
import type { Connection } from "@/lib/api/types"

type RateValues = Pick<ConnectionWithRates, "uploadRate" | "downloadRate"> & Pick<Connection, "start">
type Snapshot = { connections: readonly Connection[]; receivedAt: number }

export function useConnectionRates(
  connections: readonly Connection[],
  now: () => number = Date.now,
): ConnectionWithRates[] {
  const previousRef = useRef<Snapshot | undefined>(undefined)
  const [rates, setRates] = useState<ReadonlyMap<number, RateValues>>(() => new Map())

  useEffect(() => {
    const receivedAt = now()
    const previous = previousRef.current
    if (previous && previous.connections !== connections) {
      const elapsedMs = receivedAt - previous.receivedAt
      const rated = calculateConnectionRates(connections, previous.connections, elapsedMs)
      const next = new Map<number, RateValues>()
      for (const connection of rated) {
        if (connection.uploadRate === undefined || connection.downloadRate === undefined) continue
        next.set(connection.id, { start: connection.start, uploadRate: connection.uploadRate, downloadRate: connection.downloadRate })
      }
      setRates(next)
    }
    if (!previous || previous.connections !== connections) {
      previousRef.current = { connections, receivedAt }
    }
  }, [connections, now])

  return useMemo(() => connections.map((connection) => {
    const rate = rates.get(connection.id)
    return rate && rate.start === connection.start ? { ...connection, ...rate } : { ...connection }
  }), [connections, rates])
}
