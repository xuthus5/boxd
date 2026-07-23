import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

describe("SubscriptionsPage", () => {
  it("lists subscriptions and exposes refresh actions", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      const data = path.endsWith("/urltest-defaults")
        ? { enabled: true, url: "https://www.gstatic.com/generate_204", interval: "3m", tolerance: 50 }
        : path.endsWith("/nodes/") ? [] : [{
          id: "sub-1", name: "主订阅", url: "https://example.com/sub", interval_min: 60,
          last_updated: "2026-01-01T00:00:00Z", outbounds: [],
        }]
      return Promise.resolve(new Response(JSON.stringify(data)))
    }))
    const { container } = renderApp(<App />, "/subscriptions")

    expect(await screen.findByText("主订阅")).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "查看节点: 主订阅" })).toHaveAttribute("href", "/nodes?q=%E4%B8%BB%E8%AE%A2%E9%98%85")
    expect(screen.getByRole("link", { name: "查看日志: 主订阅" })).toHaveAttribute("href", "/observability/logs?q=%E4%B8%BB%E8%AE%A2%E9%98%85")
    expect(container.querySelector("[data-slot=card] [data-slot=card]")).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: "新增订阅" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "刷新全部" })).toBeInTheDocument()
  })

  it("treats an empty URLTest override object as global inheritance", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      const data = path.endsWith("/urltest-defaults")
        ? { enabled: true, url: "https://www.gstatic.com/generate_204", interval: "3m", tolerance: 50 }
        : path.endsWith("/nodes/") ? [] : [{
          id: "sub-1", name: "主订阅", url: "https://example.com/sub", interval_min: 60,
          last_updated: "2026-01-01T00:00:00Z", outbounds: [], urltest: {},
        }]
      return Promise.resolve(new Response(JSON.stringify(data)))
    }))
    renderApp(<App />, "/subscriptions")

    expect(await screen.findByText("URLTest：继承全局")).toBeInTheDocument()
    expect(screen.queryByText("URLTest：自定义")).not.toBeInTheDocument()
  })

  it("sorts failed subscriptions first then by last_updated", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      const data = path.endsWith("/urltest-defaults")
        ? { enabled: true, url: "https://www.gstatic.com/generate_204", interval: "3m", tolerance: 50 }
        : path.endsWith("/nodes/") ? [] : [
          { id: "old", name: "旧订阅", url: "https://example.com/old", interval_min: 60, last_updated: "2026-01-01T00:00:00Z", outbounds: [] },
          { id: "new", name: "新订阅", url: "https://example.com/new", interval_min: 60, last_updated: "2026-06-01T00:00:00Z", outbounds: [] },
          { id: "bad", name: "失败订阅", url: "https://example.com/bad", interval_min: 60, last_updated: "2026-02-01T00:00:00Z", outbounds: [], error: "timeout", error_code: "timeout" },
        ]
      return Promise.resolve(new Response(JSON.stringify(data)))
    }))
    renderApp(<App />, "/subscriptions")
    const failed = await screen.findByText("失败订阅")
    const newer = screen.getByText("新订阅")
    const older = screen.getByText("旧订阅")
    expect(failed.compareDocumentPosition(newer) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect(newer.compareDocumentPosition(older) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect(screen.getByRole("button", { name: /重试失败/ })).toBeInTheDocument()
    expect(screen.getAllByText("timeout").length).toBeGreaterThan(0)
    expect(screen.getByText("拉取超时，可稍后重试或调大网络稳定性。")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "复制错误: 失败订阅" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "复制 URL: 失败订阅" })).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "打开 URL: 失败订阅" })).toHaveAttribute("href", "https://example.com/bad")
  })
})


describe("SubscriptionsPage deep links", () => {
  function mockSubs(items: unknown[]) {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      const data = path.endsWith("/urltest-defaults")
        ? { enabled: true, url: "https://www.gstatic.com/generate_204", interval: "3m", tolerance: 50 }
        : path.endsWith("/nodes/") ? [] : items
      return Promise.resolve(new Response(JSON.stringify(data)))
    }))
  }

  it("seeds filters from deep-link query params", async () => {
    mockSubs([
      { id: "ok", name: "正常订阅", url: "https://example.com/ok", interval_min: 60, last_updated: "2026-06-01T00:00:00Z", outbounds: [] },
      { id: "bad", name: "失败订阅", url: "https://example.com/bad", interval_min: 60, last_updated: "2026-02-01T00:00:00Z", outbounds: [], error: "timeout", error_code: "timeout" },
    ])
    renderApp(<App />, "/subscriptions?status=error&q=失败")
    expect(await screen.findByLabelText("搜索订阅")).toHaveValue("失败")
    expect(screen.getByRole("button", { name: "仅失败" })).toHaveAttribute("aria-pressed", "true")
    expect(screen.getByText("失败订阅")).toBeInTheDocument()
    expect(screen.queryByText("正常订阅")).not.toBeInTheDocument()
  })

  it("updates URL filters from the toolbar", async () => {
    mockSubs([
      { id: "ok", name: "正常订阅", url: "https://example.com/ok", interval_min: 60, last_updated: "2026-06-01T00:00:00Z", outbounds: [] },
      { id: "bad", name: "失败订阅", url: "https://example.com/bad", interval_min: 60, last_updated: "2026-02-01T00:00:00Z", outbounds: [], error: "timeout", error_code: "timeout" },
    ])
    const user = userEvent.setup()
    renderApp(<App />, "/subscriptions")
    await screen.findByText("正常订阅")
    await user.click(screen.getByRole("button", { name: "仅失败" }))
    expect(screen.getByText("失败订阅")).toBeInTheDocument()
    expect(screen.queryByText("正常订阅")).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: "清除筛选" })).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "清除筛选" }))
    expect(screen.getByText("正常订阅")).toBeInTheDocument()
    expect(screen.getByText("失败订阅")).toBeInTheDocument()
  })
  it("clears filters from the empty-state action", async () => {
    mockSubs([
      { id: "ok", name: "正常订阅", url: "https://example.com/ok", interval_min: 60, last_updated: "2026-06-01T00:00:00Z", outbounds: [] },
      { id: "bad", name: "失败订阅", url: "https://example.com/bad", interval_min: 60, last_updated: "2026-02-01T00:00:00Z", outbounds: [], error: "timeout", error_code: "timeout" },
    ])
    const user = userEvent.setup()
    renderApp(<App />, "/subscriptions?q=missing-sub")

    expect(await screen.findByText("无匹配订阅")).toBeInTheDocument()
    const clearButtons = screen.getAllByRole("button", { name: "清除筛选" })
    await user.click(clearButtons[clearButtons.length - 1])
    expect(await screen.findByText("正常订阅")).toBeInTheDocument()
    expect(screen.getByText("失败订阅")).toBeInTheDocument()
  })

})
