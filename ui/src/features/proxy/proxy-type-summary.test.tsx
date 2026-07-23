import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"

import { ProxyTypeSummaryBar } from "@/features/proxy/proxy-type-summary"
import { renderApp } from "@/test/render"

describe("ProxyTypeSummaryBar", () => {
  it("renders type buckets and toggles the type filter", async () => {
    const onChange = vi.fn()
    const user = userEvent.setup()
    renderApp(
      <ProxyTypeSummaryBar
        summary={{ total: 3, buckets: [{ type: "vless", count: 2 }, { type: "direct", count: 1 }] }}
        filters={{}}
        onChange={onChange}
      />,
    )
    expect(screen.getByText(/类型分布/)).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: /vless/ }))
    expect(onChange).toHaveBeenCalledWith({ query: undefined, type: "vless" })
  })

  it("clears an active type filter on second click", async () => {
    const onChange = vi.fn()
    const user = userEvent.setup()
    renderApp(
      <ProxyTypeSummaryBar
        summary={{ total: 2, buckets: [{ type: "mixed", count: 2 }] }}
        filters={{ type: "mixed", query: "in" }}
        onChange={onChange}
      />,
    )
    await user.click(screen.getByRole("button", { name: /mixed/ }))
    expect(onChange).toHaveBeenCalledWith({ query: "in", type: undefined })
  })
})
