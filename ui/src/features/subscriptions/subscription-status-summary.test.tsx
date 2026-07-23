import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"

import { SubscriptionStatusSummaryBar } from "@/features/subscriptions/subscription-status-summary"
import { renderApp } from "@/test/render"

describe("SubscriptionStatusSummaryBar", () => {
  it("renders non-empty buckets and toggles status filters", async () => {
    const onChange = vi.fn()
    const user = userEvent.setup()
    renderApp(
      <SubscriptionStatusSummaryBar
        summary={{ total: 3, ok: 1, error: 2 }}
        filters={{}}
        onChange={onChange}
      />,
    )
    expect(screen.getByText(/状态分布/)).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: /仅失败/ }))
    expect(onChange).toHaveBeenCalledWith({ query: undefined, status: "error" })
  })

  it("clears an active status filter on second click", async () => {
    const onChange = vi.fn()
    const user = userEvent.setup()
    renderApp(
      <SubscriptionStatusSummaryBar
        summary={{ total: 2, ok: 0, error: 2 }}
        filters={{ status: "error", query: "主" }}
        onChange={onChange}
      />,
    )
    await user.click(screen.getByRole("button", { name: /仅失败/ }))
    expect(onChange).toHaveBeenCalledWith({ query: "主", status: undefined })
  })
})
