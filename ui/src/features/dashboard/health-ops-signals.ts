/** Operational health signals: failed subscriptions, unstable nodes, apply failures. */

import {
  listProblemNodes,
  summarizeNodeStability,
  type NodeHistoryMap,
  type ProblemNodePreview,
} from "@/features/nodes/nodes-filter"
import {
  failedSubscriptionIds,
  listFailedSubscriptions,
} from "@/features/subscriptions/subscription-list"
import type { ConfigApplyEvent, Outbound, Subscription } from "@/lib/api/types"

export const FAILED_SUBSCRIPTION_PREVIEW_LIMIT = 3
export const PROBLEM_NODE_PREVIEW_LIMIT = 3

export type HealthOpsSignals = {
  failedSubscriptions: number
  failedSubscriptionItems: Subscription[]
  unstableNodes: number
  failedNodes: number
  problemNodes: number
  problemNodeItems: ProblemNodePreview[]
  /** Recent config apply/validate failures (rolled_back / validate_failed). */
  applyFailures: number
  latestApplyFailure?: ConfigApplyEvent
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

export function isConfigApplyFailure(event: ConfigApplyEvent | undefined): boolean {
  if (!event) return false
  const status = event.status?.trim()
  return status === "rolled_back" || status === "validate_failed"
}

/** Count consecutive newest apply failures (stops at first success/other). */
export function summarizeApplyFailures(events: readonly ConfigApplyEvent[] | undefined): {
  applyFailures: number
  latestApplyFailure?: ConfigApplyEvent
} {
  if (!Array.isArray(events) || events.length === 0) {
    return { applyFailures: 0 }
  }
  let applyFailures = 0
  let latestApplyFailure: ConfigApplyEvent | undefined
  for (const event of events) {
    if (!isConfigApplyFailure(event)) break
    applyFailures += 1
    if (!latestApplyFailure) latestApplyFailure = event
  }
  return { applyFailures, latestApplyFailure }
}

export function buildHealthOpsSignals(input: {
  subscriptions?: readonly Subscription[]
  nodes?: readonly Outbound[]
  history?: NodeHistoryMap
  applyEvents?: readonly ConfigApplyEvent[]
  failedPreviewLimit?: number
  problemPreviewLimit?: number
}): HealthOpsSignals {
  const problems = countProblemNodes(input.nodes, input.history)
  const subscriptions = Array.isArray(input.subscriptions) ? input.subscriptions : []
  const failedLimit = input.failedPreviewLimit ?? FAILED_SUBSCRIPTION_PREVIEW_LIMIT
  const problemLimit = input.problemPreviewLimit ?? PROBLEM_NODE_PREVIEW_LIMIT
  const apply = summarizeApplyFailures(input.applyEvents)
  return {
    failedSubscriptions: countFailedSubscriptions(subscriptions),
    failedSubscriptionItems: listFailedSubscriptions(subscriptions, failedLimit),
    problemNodeItems: listProblemNodes(input.nodes, input.history, problemLimit),
    ...problems,
    ...apply,
  }
}

export function hasHealthOpsAlerts(signals: HealthOpsSignals): boolean {
  return signals.failedSubscriptions > 0 || signals.problemNodes > 0 || signals.applyFailures > 0
}
