import { screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import {
  HealthOpsAlertActions,
  HealthOpsAlertChips,
} from "@/features/dashboard/health-ops-alerts"
import { renderApp } from "@/test/render"

describe("HealthOpsAlertChips", () => {
  it("renders deep links for failed subscriptions and problem nodes", () => {
    renderApp(
      <HealthOpsAlertChips
        signals={{
          failedSubscriptions: 2,
          failedSubscriptionItems: [],
          problemNodeItems: [],
          unstableNodes: 3,
          failedNodes: 1,
          problemNodes: 4,
        }}
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
    const { container } = renderApp(
      <HealthOpsAlertChips
        signals={{
          failedSubscriptions: 0,
          failedSubscriptionItems: [],
          problemNodeItems: [],
          unstableNodes: 0,
          failedNodes: 0,
          problemNodes: 0,
        }}
      />,
    )
    expect(container.querySelector('[data-slot="health-ops-alerts"]')).toBeNull()
  })
})

describe("HealthOpsAlertActions", () => {
  it("prefers failed-node href when only failed nodes are present", () => {
    renderApp(
      <div>
        <HealthOpsAlertActions
          signals={{
            failedSubscriptions: 1,
            failedSubscriptionItems: [],
            problemNodeItems: [],
            unstableNodes: 0,
            failedNodes: 2,
            problemNodes: 2,
          }}
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
          signals={{
            failedSubscriptions: 0,
            failedSubscriptionItems: [],
            problemNodeItems: [],
            unstableNodes: 2,
            failedNodes: 1,
            problemNodes: 3,
          }}
        />
      </div>,
    )
    expect(screen.getByRole("link", { name: "查看问题节点" })).toHaveAttribute(
      "href",
      "/nodes?stability=unstable",
    )
  })
})
