import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"

import { DNSActionSummaryBar } from "@/features/policy/dns-action-summary"
import { renderApp } from "@/test/render"

describe("DNSActionSummaryBar", () => {
  it("renders action buckets and toggles the rule action filter", async () => {
    const onChange = vi.fn()
    const user = userEvent.setup()
    renderApp(
      <DNSActionSummaryBar
        summary={{ total: 3, buckets: [{ action: "route", count: 2 }, { action: "reject", count: 1 }] }}
        filters={{}}
        onChange={onChange}
      />,
    )
    expect(screen.getByText(/动作分布/)).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: /reject/ }))
    expect(onChange).toHaveBeenCalledWith({
      servers: undefined,
      rules: undefined,
      serverType: undefined,
      ruleAction: "reject",
    })
  })

  it("clears an active action filter on second click", async () => {
    const onChange = vi.fn()
    const user = userEvent.setup()
    renderApp(
      <DNSActionSummaryBar
        summary={{ total: 2, buckets: [{ action: "route", count: 2 }] }}
        filters={{ rules: "ads", ruleAction: "route", servers: "remote", serverType: "https" }}
        onChange={onChange}
      />,
    )
    await user.click(screen.getByRole("button", { name: /route/ }))
    expect(onChange).toHaveBeenCalledWith({
      servers: "remote",
      rules: "ads",
      serverType: "https",
      ruleAction: undefined,
    })
  })
})
