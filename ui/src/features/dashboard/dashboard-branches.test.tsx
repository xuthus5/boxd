import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import { HealthSummaryCard } from "@/features/dashboard/health-summary-card"
import { ServiceCard } from "@/features/dashboard/service-card"
import { TrafficChart } from "@/features/dashboard/traffic-chart"
import { calculateTrafficRates } from "@/features/dashboard/traffic-rate"
import { RecentLogs } from "@/features/dashboard/recent-logs"
import { AuthProvider } from "@/features/auth/auth-context"
import { PreferencesProvider } from "@/features/preferences/preferences-provider"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => {
  vi.unstubAllGlobals()
  sessionStore.clear()
})

function renderHealth(ui: JSX.Element) {
  sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
  vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response(JSON.stringify([])))))
  return renderApp(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <AuthProvider>
        <PreferencesProvider>
          {ui}
        </PreferencesProvider>
      </AuthProvider>
    </QueryClientProvider>,
  )
}

describe("dashboard component states", () => {
  it("enables only valid service actions for a stopped service", async () => {
    const onAction = vi.fn()
    const user = userEvent.setup()
    renderApp(<ServiceCard status={{ running: false }} onAction={onAction} />)
    expect(screen.getByText("已停止")).toBeInTheDocument()
    expect(screen.getByText("—")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "停止" })).toBeDisabled()
    expect(screen.getByRole("button", { name: /重启/ })).toBeDisabled()
    await user.click(screen.getByRole("button", { name: "启动" }))
    expect(onAction).toHaveBeenCalledWith("start")
  })

  it("disables start while the service is running", () => {
    renderApp(<ServiceCard status={{ running: true, uptime: "1m" }} onAction={vi.fn()} />)
    expect(screen.getByRole("button", { name: "启动" })).toBeDisabled()
    expect(screen.getByRole("button", { name: "停止" })).toBeEnabled()
    expect(screen.getByRole("button", { name: /重启/ })).toBeEnabled()
  })

  it("disables service actions while an operation is pending", () => {
    renderApp(<ServiceCard status={{ running: true }} pending="restart" onAction={vi.fn()} />)
    expect(screen.getByRole("button", { name: "启动" })).toBeDisabled()
    expect(screen.getByRole("button", { name: "停止" })).toBeDisabled()
    expect(screen.getByRole("button", { name: /重启/ })).toBeDisabled()
  })

  it("renders an empty traffic chart", () => {
    renderApp(<TrafficChart points={[]} />)
    expect(screen.getByText(/上传 0 B/)).toBeInTheDocument()
  })

  it("calculates traffic rates from actual sample intervals and handles resets", () => {
    expect(calculateTrafficRates([
      { timestamp: "2026-01-01T00:00:00Z", upload_bytes: 100, download_bytes: 200 },
      { timestamp: "2026-01-01T00:00:02Z", upload_bytes: 500, download_bytes: 1000 },
      { timestamp: "2026-01-01T00:00:03Z", upload_bytes: 10, download_bytes: 20 },
      { timestamp: "invalid", upload_bytes: 20, download_bytes: 30 },
    ])).toEqual([
      { timestamp: "2026-01-01T00:00:00Z", upload_rate: 0, download_rate: 0 },
      { timestamp: "2026-01-01T00:00:02Z", upload_rate: 200, download_rate: 400 },
      { timestamp: "2026-01-01T00:00:03Z", upload_rate: 0, download_rate: 0 },
      { timestamp: "invalid", upload_rate: 0, download_rate: 0 },
    ])
  })

  it("switches between real-time and cumulative traffic", async () => {
    const user = userEvent.setup()
    renderApp(<TrafficChart points={[
      { timestamp: "2026-01-01T00:00:00Z", upload_bytes: 0, download_bytes: 0 },
      { timestamp: "2026-01-01T00:00:01Z", upload_bytes: 2048, download_bytes: 4096 },
    ]} />)
    expect(screen.getByText(/上传 2.00 KB\/s/)).toBeInTheDocument()
    await user.click(screen.getByRole("tab", { name: "累计流量" }))
    expect(screen.getByText(/上传 2.00 KB · 下载 4.00 KB/)).toBeInTheDocument()
  })

  it("renders empty and populated recent logs", () => {
    const view = renderApp(<AuthProvider><PreferencesProvider><RecentLogs items={[]} /></PreferencesProvider></AuthProvider>)
    expect(screen.getByText("暂无日志")).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "查看日志" })).toHaveAttribute("href", "/observability/logs")
    view.unmount()
    renderApp(<AuthProvider><PreferencesProvider><RecentLogs items={[{ level: "error", message: "ready", timestamp: "2026-01-01T00:00:00Z" }]} /></PreferencesProvider></AuthProvider>)
    expect(screen.getByText("ready")).toBeInTheDocument()
    expect(screen.getByRole("columnheader", { name: "时间" })).toBeInTheDocument()
    expect(document.querySelector("time")).toHaveAttribute("datetime", "2026-01-01T00:00:00Z")
    expect(screen.getByText("ready").closest("td")).toHaveClass("col-span-2", "sm:table-cell")
    expect(document.querySelector("time")?.closest("td")).toHaveClass("items-center", "min-h-9")
    expect(screen.getByRole("link", { name: "查看错误日志" })).toHaveAttribute("href", "/observability/logs?preset=errors")
  })

  it("renders health summary from connection snapshot", () => {
    renderHealth(<HealthSummaryCard snapshot={{
      active_connections: 2,
      list: [
        { id: 1, target: "a.com:443", outbound: "proxy", rule: "r1", network: "tcp", upload: 10, download: 20, start: "2026-01-01T00:00:00Z" },
        { id: 2, target: "b.com:443", outbound: "proxy", rule: "r2", network: "udp", upload: 1, download: 2, start: "2026-01-01T00:00:01Z" },
      ],
    }} status={{ running: true }} />)
    expect(screen.getByText("运行健康")).toBeInTheDocument()
    expect(screen.getByText("2 条活跃连接")).toBeInTheDocument()
    expect(screen.getByText("正常")).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "查看连接" })).toHaveAttribute("href", "/observability/connections")
    expect(screen.getByRole("link", { name: "TCP 1" })).toHaveAttribute("href", "/observability/connections?network=tcp")
    expect(screen.getByRole("link", { name: "UDP 1" })).toHaveAttribute("href", "/observability/connections?network=udp")
    expect(screen.getByRole("link", { name: "proxy" })).toHaveAttribute("href", "/observability/connections?outbound=proxy")
  })
})
