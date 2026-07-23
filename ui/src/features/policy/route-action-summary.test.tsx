import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"

import { RouteActionSummaryBar } from "@/features/policy/route-action-summary"
import { renderApp } from "@/test/render"

describe("RouteActionSummaryBar", () => {
  it("renders action buckets and toggles the action filter", async () => {
    const onChange = vi.fn()
    const user = userEvent.setup()
    renderApp(
      <RouteActionSummaryBar
        summary={{ total: 3, buckets: [{ action: "route", count: 2 }, { action: "reject", count: 1 }] }}
        filters={{}}
        onChange={onChange}
      />,
    )
    expect(screen.getByText(/动作分布/)).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: /route/ }))
    expect(onChange).toHaveBeenCalledWith({ query: undefined, action: "route" })
  })

  it("clears an active action filter on second click", async () => {
    const onChange = vi.fn()
    const user = userEvent.setup()
    renderApp(
      <RouteActionSummaryBar
        summary={{ total: 2, buckets: [{ action: "reject", count: 2 }] }}
        filters={{ action: "reject", query: "ads" }}
        onChange={onChange}
      />,
    )
    await user.click(screen.getByRole("button", { name: /reject/ }))
    expect(onChange).toHaveBeenCalledWith({ query: "ads", action: undefined })
  })
})
