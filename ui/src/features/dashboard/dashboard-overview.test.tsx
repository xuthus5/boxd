import { screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import { DashboardOverview } from "@/features/dashboard/dashboard-overview"
import { renderApp } from "@/test/render"

const memory = {
  alloc: 1024,
  total: 2048,
  sys: 4096,
  num_gc: 2,
  heap_inuse: 512,
  stack_inuse: 128,
  num_goroutine: 12,
}

describe("DashboardOverview", () => {
  it("renders live rates, totals, connections, and runtime memory", () => {
    renderApp(
      <DashboardOverview
        points={[
          { upload_bytes: 1024, download_bytes: 2048, timestamp: "2026-01-01T00:00:00Z" },
          { upload_bytes: 3072, download_bytes: 6144, timestamp: "2026-01-01T00:00:01Z" },
        ]}
        activeConnections={3}
        tcpConnections={2}
        udpConnections={1}
        memory={memory}
        running
      />,
    )

    expect(screen.getByTestId("dashboard-metric-upload")).toHaveTextContent("2.00 KB/s")
    expect(screen.getByTestId("dashboard-metric-upload")).toHaveTextContent("累计 3.00 KB")
    expect(screen.getByTestId("dashboard-metric-download")).toHaveTextContent("4.00 KB/s")
    expect(screen.getByTestId("dashboard-metric-connections")).toHaveTextContent("3")
    expect(screen.getByTestId("dashboard-metric-connections")).toHaveTextContent("TCP 2 · UDP 1")
    expect(screen.getByTestId("dashboard-metric-memory")).toHaveTextContent("1.00 KB")
    expect(screen.getByTestId("dashboard-metric-memory")).toHaveTextContent("GC 2 · 12 协程")
  })

  it("falls back to zero values when no traffic or goroutine data exists", () => {
    renderApp(
      <DashboardOverview
        points={[]}
        activeConnections={0}
        tcpConnections={0}
        udpConnections={0}
        memory={{ ...memory, alloc: 0, num_goroutine: undefined }}
        running={false}
      />,
    )

    expect(screen.getByTestId("dashboard-metric-upload")).toHaveTextContent("0 B/s")
    expect(screen.getByTestId("dashboard-metric-download")).toHaveTextContent("0 B/s")
    expect(screen.getByTestId("dashboard-metric-connections")).toHaveTextContent("已停止")
    expect(screen.getByTestId("dashboard-metric-memory")).toHaveTextContent("GC 2 · 0 协程")
  })
})
