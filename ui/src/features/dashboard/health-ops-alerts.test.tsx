import { screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import {
  HealthOpsAlertActions,
  HealthOpsAlertChips,
} from "@/features/dashboard/health-ops-alerts"
import type { HealthOpsSignals } from "@/features/dashboard/health-ops-signals"
import { renderApp } from "@/test/render"

function signals(partial: Partial<HealthOpsSignals> = {}): HealthOpsSignals {
  return {
    failedSubscriptions: 0,
    failedSubscriptionItems: [],
    problemNodeItems: [],
    unstableNodes: 0,
    failedNodes: 0,
    problemNodes: 0,
    applyFailures: 0,
    ...partial,
  }
}

describe("HealthOpsAlertChips", () => {
  it("renders deep links for failed subscriptions and problem nodes", () => {
    renderApp(
      <HealthOpsAlertChips
        signals={signals({
          failedSubscriptions: 2,
          unstableNodes: 3,
          failedNodes: 1,
          problemNodes: 4,
        })}
      />,
    )
    expect(screen.getByRole("link", { name: "2 个失败订阅" })).toHaveAttribute(
      "href",
      "/subscriptions?status=error",
    )
    expect(screen.getByRole("link", { name: "3 个不稳节点" })).toHaveAttribute(
      "href",
      "/nodes?stability=unstable",
    )
    expect(screen.getByRole("link", { name: "1 个全失败节点" })).toHaveAttribute(
      "href",
      "/nodes?stability=failed",
    )
  })

  it("hides chips when there are no ops alerts", () => {
    const { container } = renderApp(<HealthOpsAlertChips signals={signals()} />)
    expect(container.querySelector('[data-slot="health-ops-alerts"]')).toBeNull()
  })

  it("renders apply failure chip with source deep-link", () => {
    renderApp(
      <HealthOpsAlertChips
        signals={signals({
          applyFailures: 2,
          latestApplyFailure: {
            id: "1",
            source: "validate_inbounds",
            status: "validate_failed",
            hash: "abc",
            size: 10,
            error: "listen_port invalid",
            applied_at: "2026-07-24T12:00:00Z",
          },
        })}
      />,
    )
    expect(screen.getByRole("link", { name: /2 次配置应用\/校验失败/ })).toHaveAttribute(
      "href",
      "/proxy/inbounds",
    )
  })
})

describe("HealthOpsAlertActions", () => {
  it("prefers failed-node href when only failed nodes are present", () => {
    renderApp(
      <div>
        <HealthOpsAlertActions
          signals={signals({
            failedSubscriptions: 1,
            failedNodes: 2,
            problemNodes: 2,
          })}
        />
      </div>,
    )
    expect(screen.getByRole("link", { name: "查看失败订阅" })).toHaveAttribute(
      "href",
      "/subscriptions?status=error",
    )
    expect(screen.getByRole("link", { name: "查看问题节点" })).toHaveAttribute(
      "href",
      "/nodes?stability=failed",
    )
  })

  it("links problem nodes to unstable when mixed issues exist", () => {
    renderApp(
      <div>
        <HealthOpsAlertActions
          signals={signals({
            unstableNodes: 2,
            failedNodes: 1,
            problemNodes: 3,
          })}
        />
      </div>,
    )
    expect(screen.getByRole("link", { name: "查看问题节点" })).toHaveAttribute(
      "href",
      "/nodes?stability=unstable",
    )
  })

  it("links apply failure action to editor source", () => {
    renderApp(
      <div>
        <HealthOpsAlertActions
          signals={signals({
            applyFailures: 1,
            latestApplyFailure: {
              id: "1",
              source: "validate_dns",
              status: "validate_failed",
              hash: "abc",
              size: 10,
              applied_at: "2026-07-24T12:00:00Z",
            },
          })}
        />
      </div>,
    )
    expect(screen.getByRole("link", { name: "打开失败来源" })).toHaveAttribute(
      "href",
      "/policy/dns",
    )
  })
})
