import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"

import { DNSTypeSummaryBar } from "@/features/policy/dns-type-summary"
import { renderApp } from "@/test/render"

describe("DNSTypeSummaryBar", () => {
  it("renders type buckets and toggles the server type filter", async () => {
    const onChange = vi.fn()
    const user = userEvent.setup()
    renderApp(
      <DNSTypeSummaryBar
        summary={{ total: 3, buckets: [{ type: "https", count: 2 }, { type: "udp", count: 1 }] }}
        filters={{}}
        onChange={onChange}
      />,
    )
    expect(screen.getByText(/类型分布/)).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: /https/ }))
    expect(onChange).toHaveBeenCalledWith({
      servers: undefined,
      rules: undefined,
      serverType: "https",
      ruleAction: undefined,
    })
  })

  it("clears an active type filter on second click", async () => {
    const onChange = vi.fn()
    const user = userEvent.setup()
    renderApp(
      <DNSTypeSummaryBar
        summary={{ total: 2, buckets: [{ type: "https", count: 2 }] }}
        filters={{ servers: "remote", serverType: "https", rules: "ads", ruleAction: "reject" }}
        onChange={onChange}
      />,
    )
    await user.click(screen.getByRole("button", { name: /https/ }))
    expect(onChange).toHaveBeenCalledWith({
      servers: "remote",
      rules: "ads",
      serverType: undefined,
      ruleAction: "reject",
    })
  })
})
