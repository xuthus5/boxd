import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"

import { NodeStabilitySummaryBar } from "@/features/nodes/node-stability-summary"
import { renderApp } from "@/test/render"

describe("NodeStabilitySummaryBar", () => {
  it("renders non-empty buckets and toggles stability filters", async () => {
    const onChange = vi.fn()
    const user = userEvent.setup()
    renderApp(
      <NodeStabilitySummaryBar
        summary={{ total: 5, stable: 2, fair: 1, unstable: 1, failed: 1, unknown: 0 }}
        filters={{}}
        onChange={onChange}
      />,
    )
    expect(screen.getByText(/稳定性分布/)).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /未知/ })).not.toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: /稳定/ }))
    expect(onChange).toHaveBeenCalledWith({ stability: "stable", query: undefined, sort: undefined })
  })

  it("clears an active stability filter on second click", async () => {
    const onChange = vi.fn()
    const user = userEvent.setup()
    renderApp(
      <NodeStabilitySummaryBar
        summary={{ total: 2, stable: 2, fair: 0, unstable: 0, failed: 0, unknown: 0 }}
        filters={{ stability: "stable", sort: "latency" }}
        onChange={onChange}
      />,
    )
    await user.click(screen.getByRole("button", { name: /稳定/ }))
    expect(onChange).toHaveBeenCalledWith({ stability: undefined, query: undefined, sort: "latency" })
  })
})
