import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { screen, waitFor } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { KernelStatusBadge } from "@/features/dashboard/kernel-status-badge"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

function renderBadge() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return renderApp(
    <QueryClientProvider client={client}>
      <KernelStatusBadge />
    </QueryClientProvider>,
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
  sessionStore.clear()
})

describe("KernelStatusBadge", () => {
  it("shows running status with uptime and links to dashboard", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response(JSON.stringify({
      running: true,
      uptime: "12m",
    })))))
    renderBadge()
    const link = await screen.findByRole("link", { name: "运行中 · 12m" })
    expect(link).toHaveAttribute("href", "/dashboard")
    expect(document.querySelector('[data-kernel-status="running"]')).toBeInTheDocument()
  })

  it("shows stopped status", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response(JSON.stringify({
      running: false,
    })))))
    renderBadge()
    expect(await screen.findByRole("link", { name: "已停止" })).toHaveAttribute("href", "/dashboard")
    expect(document.querySelector('[data-kernel-status="stopped"]')).toBeInTheDocument()
  })

  it("shows failed status when the last start error is present", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response(JSON.stringify({
      running: false,
      last_error: "invalid outbound",
    })))))
    renderBadge()
    const link = await screen.findByRole("link", { name: "启动失败: invalid outbound" })
    expect(link).toHaveAttribute("href", "/dashboard")
    expect(document.querySelector('[data-kernel-status="failed"]')).toBeInTheDocument()
  })

  it("shows unknown status when the request fails", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response(JSON.stringify({
      code: "internal_error",
      message: "boom",
    }), { status: 500 }))))
    renderBadge()
    await waitFor(() => {
      expect(screen.getByRole("link", { name: "状态未知" })).toBeInTheDocument()
    })
    expect(document.querySelector('[data-kernel-status="unknown"]')).toBeInTheDocument()
  })
})
