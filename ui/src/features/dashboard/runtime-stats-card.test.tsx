import { screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import { RuntimeStatsCard } from "@/features/dashboard/runtime-stats-card"
import { renderApp } from "@/test/render"

describe("RuntimeStatsCard", () => {
  it("renders dense memory and version metrics", () => {
    renderApp(
      <RuntimeStatsCard
        memory={{
          alloc: 1024,
          total: 2048,
          sys: 4096,
          num_gc: 2,
          heap_inuse: 512,
          stack_inuse: 128,
          num_goroutine: 12,
        }}
        panelVersion="dev"
        kernelVersion="1.13.14"
      />,
    )
    expect(screen.getByText("运行时统计")).toBeInTheDocument()
    expect(screen.getByText("1.00 KB")).toBeInTheDocument()
    expect(screen.getByText("4.00 KB")).toBeInTheDocument()
    expect(screen.getByText("512 B")).toBeInTheDocument()
    expect(screen.getByText("128 B")).toBeInTheDocument()
    expect(screen.getByText("2")).toBeInTheDocument()
    expect(screen.getByText("12")).toBeInTheDocument()
    expect(screen.getByText("1.13.14")).toBeInTheDocument()
    expect(screen.getByText("dev")).toBeInTheDocument()
  })

  it("falls back when goroutine count is missing", () => {
    renderApp(
      <RuntimeStatsCard
        memory={{
          alloc: 0,
          total: 0,
          sys: 0,
          num_gc: 0,
          heap_inuse: 0,
          stack_inuse: 0,
        }}
        panelVersion=""
        kernelVersion=""
      />,
    )
    expect(screen.getAllByText("—").length).toBeGreaterThanOrEqual(2)
  })
})
