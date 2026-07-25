import { fireEvent, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { toast } from "sonner"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

vi.mock("sonner", async (importOriginal) => {
  const actual = await importOriginal<typeof import("sonner")>()
  return { ...actual, toast: { success: vi.fn(), error: vi.fn() } }
})

const urlTestDefaults = { enabled: true, url: "https://www.gstatic.com/generate_204", interval: "3m", tolerance: 50 }

function installSubscriptionsActionAPI(
  items: unknown[],
  respond: (path: string, method: string) => Response | Promise<Response> | undefined,
) {
  const fetchMock = vi.fn((input: string | URL | Request, init?: RequestInit) => {
    const rawURL = typeof input === "string" ? input : input instanceof URL ? input.toString() : input.url
    const path = new URL(rawURL, "http://localhost").pathname
    const method = input instanceof Request ? input.method : init?.method ?? "GET"
    const custom = respond(path, method)
    if (custom) return Promise.resolve(custom)
    const data = path.endsWith("/urltest-defaults") ? urlTestDefaults : path.endsWith("/nodes/") ? [] : items
    return Promise.resolve(new Response(JSON.stringify(data)))
  })
  vi.stubGlobal("fetch", fetchMock)
  return fetchMock
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.clearAllMocks()
  sessionStore.clear()
})

describe("SubscriptionsPage actions", () => {
  it("reports individual refresh failures, retries, and refreshes all", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    let refreshCalls = 0
    installSubscriptionsActionAPI([
      {
        id: "sub-1", name: "主订阅", url: "https://example.com/sub", interval_min: 60,
        last_updated: "2026-01-01T00:00:00Z", outbounds: [],
      },
      {
        id: "sub-2", name: "失败订阅", url: "https://example.com/bad", interval_min: 60,
        last_updated: "2026-01-01T00:00:00Z", outbounds: [], error: "timeout",
      },
    ], (path, method) => {
      if (path === "/api/subscriptions/sub-1/refresh" && method === "POST") {
        refreshCalls += 1
        if (refreshCalls === 1) {
          return new Response(JSON.stringify({ status: "error", data: null, error: { code: "subscription_refresh_failed", message: "timeout" }, meta: null }), { status: 500 })
        }
        return new Response(JSON.stringify({}))
      }
      if (path === "/api/subscriptions/sub-2/refresh" && method === "POST") {
        return new Response(JSON.stringify({}))
      }
      if (path === "/api/subscriptions/refresh-all" && method === "POST") {
        return new Response(JSON.stringify({ status: "ok", data: null, error: null, meta: null }))
      }
      return undefined
    })
    const user = userEvent.setup()
    renderApp(<App />, "/subscriptions")

    await screen.findByText("主订阅")
    await user.click(screen.getByRole("button", { name: "刷新" }))
    await waitFor(() => expect(toast.error).toHaveBeenCalled())
    await user.click(screen.getByRole("button", { name: "刷新" }))
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith("订阅已刷新"))
    await user.click(screen.getByRole("button", { name: "重试刷新" }))
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith("已重试刷新"))
    await user.click(screen.getByRole("button", { name: "刷新全部" }))
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith("全部订阅已刷新"))
  })

  it("locks batch actions while refresh-all is pending", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    let refreshAllCalls = 0
    let resolveRefreshAll!: (response: Response) => void
    const pendingRefreshAll = new Promise<Response>((resolve) => { resolveRefreshAll = resolve })
    installSubscriptionsActionAPI([{
      id: "sub-1", name: "失败订阅", url: "https://example.com/sub", interval_min: 60,
      last_updated: "2026-01-01T00:00:00Z", outbounds: [], error: "timeout",
    }], (path, method) => {
      if (path !== "/api/subscriptions/refresh-all" || method !== "POST") return undefined
      refreshAllCalls += 1
      return pendingRefreshAll
    })
    const user = userEvent.setup()
    renderApp(<App />, "/subscriptions")

    const refreshAllButton = await screen.findByRole("button", { name: "刷新全部" })
    const retryButton = screen.getByRole("button", { name: "重试失败 (1)" })
    await user.click(refreshAllButton)
    expect(refreshAllButton).toBeDisabled()
    expect(retryButton).toBeDisabled()
    expect(screen.getByRole("button", { name: "刷新中" })).toBeDisabled()
    refreshAllButton.removeAttribute("disabled")
    await user.click(refreshAllButton)
    expect(refreshAllCalls).toBe(1)

    resolveRefreshAll(new Response(JSON.stringify({ status: "ok", data: null, error: null, meta: null })))
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith("全部订阅已刷新"))
    expect(screen.getByRole("button", { name: "刷新全部" })).not.toBeDisabled()
  })

  it("reports partial refresh-all details and fallback counts", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    let refreshAllCalls = 0
    installSubscriptionsActionAPI([], (path, method) => {
      if (path !== "/api/subscriptions/refresh-all" || method !== "POST") return undefined
      refreshAllCalls += 1
      const data = refreshAllCalls === 1
        ? { failed: [{ id: "bad", name: "失败订阅", code: "timeout", message: "timeout" }], sync_error: "restart failed" }
        : { failed: [] }
      const error = refreshAllCalls === 1
        ? { code: "subscription_refresh_failed", message: "partial" }
        : null
      const meta = refreshAllCalls === 1 ? { failed_count: 2 } : null
      return new Response(JSON.stringify({ status: "partial", data, error, meta }))
    })
    const user = userEvent.setup()
    renderApp(<App />, "/subscriptions")

    await user.click(await screen.findByRole("button", { name: "刷新全部" }))
    await waitFor(() => expect(toast.error).toHaveBeenCalledTimes(1))
    expect(String(vi.mocked(toast.error).mock.calls[0][0])).toContain("配置同步")
    await user.click(screen.getByRole("button", { name: "刷新全部" }))
    await waitFor(() => expect(toast.error).toHaveBeenCalledTimes(2))
    expect(String(vi.mocked(toast.error).mock.calls[1][0])).toContain("1 个刷新/同步操作失败")
  })

  it("retries failed subscriptions and keeps per-item diagnostics", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const items = [
      { id: "ok", name: "恢复订阅", url: "https://example.com/ok", interval_min: 60, last_updated: "2026-01-01T00:00:00Z", outbounds: [], error: "timeout" },
      { id: "bad", name: "仍失败订阅", url: "https://example.com/bad", interval_min: 60, last_updated: "2026-01-01T00:00:00Z", outbounds: [], error: "timeout" },
    ]
    let badRefreshCalls = 0
    installSubscriptionsActionAPI(items, (path, method) => {
      if (path === "/api/subscriptions/ok/refresh" && method === "POST") return new Response(JSON.stringify({}))
      if (path === "/api/subscriptions/bad/refresh" && method === "POST") {
        badRefreshCalls += 1
        if (badRefreshCalls === 1) {
          return new Response(JSON.stringify({ status: "error", data: null, error: { code: "subscription_refresh_failed", message: "subscription HTTP 403" }, meta: null }), { status: 500 })
        }
        return new Response(JSON.stringify({}))
      }
      return undefined
    })
    const user = userEvent.setup()
    renderApp(<App />, "/subscriptions")

    await user.click(await screen.findByRole("button", { name: "重试失败 (2)" }))
    await waitFor(() => expect(toast.error).toHaveBeenCalled())
    expect(vi.mocked(toast.error).mock.calls.at(-1)?.[0]).toBe("重试完成：成功 1，失败 1")
    expect(screen.getByRole("button", { name: "仅失败" })).toHaveAttribute("aria-pressed", "true")
    const retryButton = screen.getByRole("button", { name: "重试失败 (2)" })
    await waitFor(() => expect(retryButton).not.toBeDisabled())
    await user.click(retryButton)
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith("已重试 2 个失败订阅"))
  })

  it("creates a subscription from an empty list", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const fetchMock = installSubscriptionsActionAPI([], (path, method) => {
      if (path === "/api/subscriptions/" && method === "POST") {
        return new Response(JSON.stringify({ id: "sub-new" }), { status: 201 })
      }
      if (path === "/api/subscriptions/sub-new/refresh" && method === "POST") {
        return new Response(JSON.stringify({}))
      }
      return undefined
    })
    const user = userEvent.setup()
    renderApp(<App />, "/subscriptions")

    await user.click(await screen.findByRole("button", { name: "新增订阅" }))
    fireEvent.change(screen.getByLabelText("名称"), { target: { value: "新订阅" } })
    fireEvent.change(screen.getByLabelText("订阅 URL"), { target: { value: "https://example.com/new" } })
    await user.click(screen.getByRole("button", { name: "保存" }))
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith("订阅已保存"))
    expect(fetchMock).toHaveBeenCalledWith("/api/subscriptions/sub-new/refresh", expect.objectContaining({ method: "POST" }))
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
  })

  it("closes after a persisted create when the initial refresh fails", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    installSubscriptionsActionAPI([], (path, method) => {
      if (path === "/api/subscriptions/" && method === "POST") {
        return new Response(JSON.stringify({ id: "sub-new" }), { status: 201 })
      }
      if (path === "/api/subscriptions/sub-new/refresh" && method === "POST") {
        return new Response(JSON.stringify({
          status: "error",
          data: null,
          error: { code: "subscription_refresh_failed", message: "timeout" },
          meta: null,
        }), { status: 500 })
      }
      return undefined
    })
    const user = userEvent.setup()
    renderApp(<App />, "/subscriptions")

    await user.click(await screen.findByRole("button", { name: "新增订阅" }))
    fireEvent.change(screen.getByLabelText("名称"), { target: { value: "新订阅" } })
    fireEvent.change(screen.getByLabelText("订阅 URL"), { target: { value: "https://example.com/new" } })
    await user.click(screen.getByRole("button", { name: "保存" }))

    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
    expect(String(vi.mocked(toast.error).mock.calls.at(-1)?.[0])).toContain("已保存，但刷新失败")
  })

  it("refreshes changed URLs and only syncs unchanged subscriptions", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const fetchMock = installSubscriptionsActionAPI([{
      id: "sub-1", name: "主订阅", url: "https://example.com/sub", interval_min: 60,
      last_updated: "2026-01-01T00:00:00Z", outbounds: [],
    }], () => undefined)
    const user = userEvent.setup()
    renderApp(<App />, "/subscriptions")

    await user.click(await screen.findByRole("button", { name: "编辑" }))
    await user.click(screen.getByRole("button", { name: "保存" }))
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
    expect(fetchMock).toHaveBeenCalledWith("/api/nodes/sync-config", expect.objectContaining({ method: "POST" }))
    expect(fetchMock.mock.calls.some(([input]) => String(input).includes("/sub-1/refresh"))).toBe(false)

    await user.click(screen.getByRole("button", { name: "编辑" }))
    await user.clear(screen.getByLabelText("订阅 URL"))
    await user.type(screen.getByLabelText("订阅 URL"), "https://example.com/changed")
    await user.click(screen.getByRole("button", { name: "保存" }))
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
    expect(fetchMock).toHaveBeenCalledWith("/api/subscriptions/sub-1/refresh", expect.objectContaining({ method: "POST" }))
  })

  it("edits and deletes a subscription", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    installSubscriptionsActionAPI([{
      id: "sub-1", name: "主订阅", url: "https://example.com/sub", interval_min: 60,
      last_updated: "2026-01-01T00:00:00Z", outbounds: [],
    }], () => undefined)
    const user = userEvent.setup()
    renderApp(<App />, "/subscriptions")

    await user.click(await screen.findByRole("button", { name: "编辑" }))
    expect(screen.getByRole("heading", { name: "编辑订阅" })).toBeInTheDocument()
    expect(screen.getByLabelText("名称")).toHaveValue("主订阅")
    await user.click(screen.getByRole("button", { name: "取消" }))
    await user.click(screen.getByRole("button", { name: "删除" }))
    await user.click(screen.getByRole("button", { name: "确认删除" }))
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith("订阅已删除"))
  })
})
