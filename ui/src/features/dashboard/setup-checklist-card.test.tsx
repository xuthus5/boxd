import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import { SetupChecklistCard } from "@/features/dashboard/setup-checklist-card"
import type { SetupStatus } from "@/lib/api/setup"
import { renderApp } from "@/test/render"
import { setupStatus } from "@/test/setup-fixtures"

afterEach(() => vi.unstubAllGlobals())

function readyStatus(): SetupStatus {
  const value = setupStatus()
  return { ...value, modules: value.modules.map((item) => ({ ...item, state: "ready", count: 1 })),
    proxy_ready: true, kernel_running: true,
    listeners: [{ tag: "mixed-in", type: "mixed", listen: "0.0.0.0", listen_port: 1080 }] }
}

function renderCard(input: { setup?: SetupStatus; subscriptions?: unknown[]; fail?: string } = {}) {
  const fetchMock = vi.fn(async (url: string) => {
    if (url.includes(input.fail ?? "never-match")) return new Response(JSON.stringify({ error: "boom" }), { status: 500 })
    return new Response(JSON.stringify(url === "/api/config/setup" ? input.setup ?? readyStatus() : input.subscriptions ?? []))
  })
  vi.stubGlobal("fetch", fetchMock)
  const view = renderApp(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
    <SetupChecklistCard />
  </QueryClientProvider>)
  return { ...view, fetchMock }
}

describe("SetupChecklistCard", () => {
  it("shows ordered steps and completes node setup without a subscription", async () => {
    const view = renderCard()
    expect(await screen.findByText("4 / 4")).toBeInTheDocument()
    expect([...view.container.querySelectorAll("[data-setup-step]")].map((item) => item.getAttribute("data-setup-step")))
      .toEqual(["modules", "nodes", "access", "kernel"])
    expect(screen.getByRole("button", { name: "管理/初始化模块" })).toBeEnabled()
    expect(screen.getByRole("link", { name: "导入节点或订阅" })).toHaveAttribute("href", "/subscriptions")
    expect(screen.getByRole("button", { name: "重新校验" })).toBeEnabled()
  })

  it("keeps failed subscriptions visible without treating an empty subscription as a node", async () => {
    const status = readyStatus()
    status.proxy_ready = false
    status.kernel_running = false
    renderCard({ setup: status, subscriptions: [{ id: "bad", name: "失败订阅", url: "https://example.com/sub", interval_min: 60,
      last_updated: "2026-01-01T00:00:00Z", outbounds: [], error: "timeout" }] })
    expect(await screen.findByText("1 个订阅刷新失败")).toBeInTheDocument()
    expect(screen.getByText("2 / 4")).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "查看失败订阅" })).toHaveAttribute("href", "/subscriptions?status=error")
    expect(screen.getByRole("button", { name: "校验并启动" })).toBeDisabled()
  })

  it("opens module management and connection instructions after setup", async () => {
    renderCard()
    const user = userEvent.setup()
    await user.click(await screen.findByRole("button", { name: "管理/初始化模块" }))
    expect(await screen.findByRole("dialog", { name: "模块化配置" })).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "取消" }))
    await user.click(screen.getByRole("button", { name: "接入说明" }))
    expect(await screen.findByText("-p 127.0.0.1:1080:1080")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "接入说明" }))
    expect(screen.queryByText("-p 127.0.0.1:1080:1080")).not.toBeInTheDocument()
  })

  it.each(["/api/config/setup", "/api/subscriptions"])("reports and retries %s load failures", async (fail) => {
    const { fetchMock } = renderCard({ fail })
    const user = userEvent.setup()
    expect(await screen.findByTestId("card-query-error")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "重试" }))
    await waitFor(() => expect(fetchMock.mock.calls.filter(([url]) => url.includes(fail)).length).toBe(2))
  })
})
