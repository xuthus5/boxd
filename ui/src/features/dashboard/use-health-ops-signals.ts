/** Load subscription and node signals for dashboard health card. */

import { useQuery } from "@tanstack/react-query"
import { useMemo } from "react"

import {
  buildHealthOpsSignals,
  type HealthOpsSignals,
} from "@/features/dashboard/health-ops-signals"
import { api } from "@/lib/api/endpoints"
import type { LatencyPoint, Subscription } from "@/lib/api/types"

export function useHealthOpsSignals(): HealthOpsSignals {
  const subscriptions = useQuery({ queryKey: ["subscriptions"], queryFn: api.subscriptions.list })
  const nodes = useQuery({ queryKey: ["nodes"], queryFn: api.nodes.list })
  const history = useQuery({
    queryKey: ["nodes", "history"],
    queryFn: async () => {
      const payload = await api.nodes.testHistory()
      return (payload.history ?? {}) as Record<string, Record<string, LatencyPoint[]>>
    },
  })
  return useMemo(
    () => buildHealthOpsSignals({
      subscriptions: Array.isArray(subscriptions.data)
        ? subscriptions.data as Subscription[]
        : undefined,
      nodes: nodes.data,
      history: history.data,
    }),
    [history.data, nodes.data, subscriptions.data],
  )
}
