/** Operational health signals: failed subscriptions and unstable nodes. */

import {
  summarizeNodeStability,
  type NodeHistoryMap,
} from "@/features/nodes/nodes-filter"
import {
  failedSubscriptionIds,
  listFailedSubscriptions,
} from "@/features/subscriptions/subscription-list"
import type { Outbound, Subscription } from "@/lib/api/types"

export const FAILED_SUBSCRIPTION_PREVIEW_LIMIT = 3

export type HealthOpsSignals = {
  failedSubscriptions: number
  failedSubscriptionItems: Subscription[]
  unstableNodes: number
  failedNodes: number
  problemNodes: number
}

export function countFailedSubscriptions(items: readonly Subscription[] | undefined): number {
  if (!Array.isArray(items) || items.length === 0) return 0
  return failedSubscriptionIds([...items]).length
}

export function countProblemNodes(
  nodes: readonly Outbound[] | undefined,
  history: NodeHistoryMap | undefined,
): Pick<HealthOpsSignals, "unstableNodes" | "failedNodes" | "problemNodes"> {
  if (!Array.isArray(nodes) || nodes.length === 0) {
    return { unstableNodes: 0, failedNodes: 0, problemNodes: 0 }
  }
  const summary = summarizeNodeStability(nodes, history)
  const unstableNodes = summary.unstable
  const failedNodes = summary.failed
  return {
    unstableNodes,
    failedNodes,
    problemNodes: unstableNodes + failedNodes,
  }
}

export function buildHealthOpsSignals(input: {
  subscriptions?: readonly Subscription[]
  nodes?: readonly Outbound[]
  history?: NodeHistoryMap
  failedPreviewLimit?: number
}): HealthOpsSignals {
  const problems = countProblemNodes(input.nodes, input.history)
  const subscriptions = Array.isArray(input.subscriptions) ? input.subscriptions : []
  const limit = input.failedPreviewLimit ?? FAILED_SUBSCRIPTION_PREVIEW_LIMIT
  return {
    failedSubscriptions: countFailedSubscriptions(subscriptions),
    failedSubscriptionItems: listFailedSubscriptions(subscriptions, limit),
    ...problems,
  }
}

export function hasHealthOpsAlerts(signals: HealthOpsSignals): boolean {
  return signals.failedSubscriptions > 0 || signals.problemNodes > 0
}
