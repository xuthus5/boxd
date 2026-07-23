import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"

import { ConnectionFacetSummaryBar } from "@/features/observability/connection-facet-summary"
import { renderApp } from "@/test/render"

describe("ConnectionFacetSummaryBar", () => {
  it("renders scoped facet chips and toggles a field filter", async () => {
    const onChange = vi.fn()
    const user = userEvent.setup()
    renderApp(
      <ConnectionFacetSummaryBar
        sections={[
          {
            field: "network",
            options: [
              { value: "tcp", count: 3 },
              { value: "udp", count: 1 },
            ],
          },
          {
            field: "outbound",
            options: [{ value: "proxy", count: 2 }],
          },
        ]}
        filters={{}}
        onChange={onChange}
      />,
    )
    expect(screen.getByText("分布概览")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: /tcp/ }))
    expect(onChange).toHaveBeenCalledWith({ network: "tcp" })
  })

  it("clears an active chip on second click", async () => {
    const onChange = vi.fn()
    const user = userEvent.setup()
    renderApp(
      <ConnectionFacetSummaryBar
        sections={[
          {
            field: "protocol",
            options: [{ value: "tls", count: 2 }],
          },
        ]}
        filters={{ protocol: "tls" }}
        onChange={onChange}
      />,
    )
    await user.click(screen.getByRole("button", { name: /tls/ }))
    expect(onChange).toHaveBeenCalledWith({ protocol: "" })
  })
})
