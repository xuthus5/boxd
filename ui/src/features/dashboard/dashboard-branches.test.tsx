import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"

import { ServiceCard } from "@/features/dashboard/service-card"
import { TrafficChart } from "@/features/dashboard/traffic-chart"
import { RecentLogs } from "@/features/dashboard/recent-logs"
import { renderApp } from "@/test/render"

describe("dashboard component states", () => {
  it("shows a stopped service and pending action", async () => {
    const onAction = vi.fn()
    const user = userEvent.setup()
    renderApp(<ServiceCard status={{ running: false }} pending="restart" onAction={onAction} />)
    expect(screen.getByText("已停止")).toBeInTheDocument()
    expect(screen.getByText("—")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /重启/ })).toBeDisabled()
    await user.click(screen.getByRole("button", { name: "启动" }))
    expect(onAction).not.toHaveBeenCalled()
  })

  it("renders an empty traffic chart", () => {
    renderApp(<TrafficChart points={[]} />)
    expect(screen.getByText(/上传 0 B/)).toBeInTheDocument()
  })

  it("renders empty and populated recent logs", () => {
    const view = renderApp(<RecentLogs items={[]} />)
    expect(screen.getByText("暂无日志")).toBeInTheDocument()
    view.unmount()
    renderApp(<RecentLogs items={[{ level: "error", message: "ready", timestamp: "2026-01-01T00:00:00Z" }]} />)
    expect(screen.getByText("ready")).toBeInTheDocument()
    expect(screen.getByRole("columnheader", { name: "时间" })).toBeInTheDocument()
    expect(document.querySelector("time")).toHaveAttribute("datetime", "2026-01-01T00:00:00Z")
  })
})
