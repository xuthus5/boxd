import { screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import { LatencyHealthBar } from "@/features/nodes/latency-health-bar"
import { renderApp } from "@/test/render"

describe("LatencyHealthBar", () => {
  it("renders empty and populated success rates", () => {
    const empty = renderApp(<LatencyHealthBar points={[]} />)
    expect(screen.getByText("未知")).toBeInTheDocument()
    expect(screen.getByText("暂无样本")).toBeInTheDocument()
    expect(screen.getByRole("progressbar")).toHaveAttribute("aria-valuenow", "0")
    empty.unmount()

    renderApp(<LatencyHealthBar points={[
      { timestamp: "1", success: true, latency_ms: 20 },
      { timestamp: "2", success: false, error: "timeout" },
      { timestamp: "3", success: true, latency_ms: 40 },
      { timestamp: "4", success: true, latency_ms: 30 },
      { timestamp: "5", success: true, latency_ms: 25 },
    ]} />)
    expect(screen.getByText("稳定")).toBeInTheDocument()
    expect(screen.getByText("成功率 80%（4/5）")).toBeInTheDocument()
    expect(screen.getByRole("progressbar")).toHaveAttribute("aria-valuenow", "80")
  })
})
