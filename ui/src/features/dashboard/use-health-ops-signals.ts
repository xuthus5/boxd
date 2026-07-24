/** Load subscription, node, and apply signals for dashboard health card. */

import { useQuery } from "@tanstack/react-query"
import { useMemo } from "react"

import {
  buildHealthOpsSignals,
  type HealthOpsSignals,
} from "@/features/dashboard/health-ops-signals"
import { api } from "@/lib/api/endpoints"
import type { ConfigApplyEvent, LatencyPoint, Subscription } from "@/lib/api/types"

export type HealthOpsQueryState = {
  signals: HealthOpsSignals
  queryError?: unknown
  queryScope?: string
  onRetry?: () => void
}

export function useHealthOpsSignals(): HealthOpsQueryState {
  const subscriptions = useQuery({ queryKey: ["subscriptions"], queryFn: api.subscriptions.list })
  const nodes = useQuery({ queryKey: ["nodes"], queryFn: api.nodes.list })
  const history = useQuery({
    queryKey: ["nodes", "history"],
    queryFn: async () => {
      const payload = await api.nodes.testHistory()
      return (payload.history ?? {}) as Record<string, Record<string, LatencyPoint[]>>
    },
  })
  const applyHistory = useQuery({
    queryKey: ["config", "apply-history"],
    queryFn: api.config.applyHistory,
    refetchInterval: 15000,
  })
  const signals = useMemo(
    () => buildHealthOpsSignals({
      subscriptions: Array.isArray(subscriptions.data)
        ? subscriptions.data as Subscription[]
        : undefined,
      nodes: nodes.data,
      history: history.data,
      applyEvents: Array.isArray(applyHistory.data?.events)
        ? applyHistory.data.events as ConfigApplyEvent[]
        : undefined,
    }),
    [applyHistory.data?.events, history.data, nodes.data, subscriptions.data],
  )
  const queryError = subscriptions.error
    || nodes.error
    || history.error
    || applyHistory.error
    || undefined
  const queryScope = subscriptions.error
    ? "health-subscriptions"
    : nodes.error
      ? "health-nodes"
      : history.error
        ? "health-node-history"
        : applyHistory.error
          ? "health-apply-history"
          : undefined
  const onRetry = queryError
    ? () => {
      if (subscriptions.error) void subscriptions.refetch()
      if (nodes.error) void nodes.refetch()
      if (history.error) void history.refetch()
      if (applyHistory.error) void applyHistory.refetch()
    }
    : undefined
  return { signals, queryError, queryScope, onRetry }
}
