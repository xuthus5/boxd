import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"

import { LogLevelSummaryBar } from "@/features/observability/log-level-summary"
import { renderApp } from "@/test/render"

describe("LogLevelSummaryBar", () => {
  it("renders level buckets and toggles the level filter", async () => {
    const onChange = vi.fn()
    const user = userEvent.setup()
    renderApp(
      <LogLevelSummaryBar
        summary={{ total: 3, buckets: [{ level: "error", count: 2 }, { level: "info", count: 1 }] }}
        filters={{}}
        onChange={onChange}
      />,
    )
    expect(screen.getByText(/级别分布/)).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: /error/ }))
    expect(onChange).toHaveBeenCalledWith({ level: "error" })
  })

  it("clears an active level filter on second click", async () => {
    const onChange = vi.fn()
    const user = userEvent.setup()
    renderApp(
      <LogLevelSummaryBar
        summary={{ total: 2, buckets: [{ level: "warn", count: 2 }] }}
        filters={{ level: "warn" }}
        onChange={onChange}
      />,
    )
    await user.click(screen.getByRole("button", { name: /warn/ }))
    expect(onChange).toHaveBeenCalledWith({ level: undefined })
  })
})
