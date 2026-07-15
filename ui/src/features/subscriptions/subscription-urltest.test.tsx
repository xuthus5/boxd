import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

const defaults = {
  enabled: true,
  url: "https://www.gstatic.com/generate_204",
  interval: "3m",
  tolerance: 50,
}

function installSubscriptionAPI(subscriptions: unknown[] = []) {
  const fetchMock = vi.fn((input: string | URL | Request, init?: RequestInit) => {
    const path = typeof input === "string" ? input : input.toString()
    if (path.endsWith("/api/subscriptions/") && !init?.method) {
      return Promise.resolve(new Response(JSON.stringify(subscriptions)))
    }
    if (path.endsWith("/api/settings/urltest-defaults")) {
      return Promise.resolve(new Response(JSON.stringify(defaults)))
    }
    if (path.endsWith("/api/nodes/")) {
      return Promise.resolve(new Response(JSON.stringify([])))
    }
    return Promise.resolve(new Response("{}"))
  })
  vi.stubGlobal("fetch", fetchMock)
  return fetchMock
}

describe("subscription URLTest overrides", () => {
  it("submits only explicit URLTest overrides", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const fetchMock = installSubscriptionAPI()
    const user = userEvent.setup()
    renderApp(<App />, "/subscriptions")

    await user.click(await screen.findByRole("button", { name: "新增订阅" }))
    expect(screen.getByLabelText("URLTest 测试地址")).toHaveAttribute("placeholder", defaults.url)
    expect(screen.getByLabelText("URLTest 测试间隔")).toHaveAttribute("placeholder", defaults.interval)
    await user.type(screen.getByLabelText("名称"), "主订阅")
    await user.type(screen.getByLabelText("订阅 URL"), "https://example.com/sub")
    await user.click(screen.getByRole("button", { name: "启用" }))
    await user.type(screen.getByLabelText("URLTest 测试间隔"), "5m")
    await user.type(screen.getByLabelText("URLTest 切换容差（毫秒）"), "0")
    await user.click(screen.getByRole("button", { name: "保存" }))

    expect(fetchMock).toHaveBeenCalledWith("/api/subscriptions/", expect.objectContaining({
      method: "POST",
      body: JSON.stringify({
        name: "主订阅",
        url: "https://example.com/sub",
        interval_min: 60,
        urltest: { enabled: true, interval: "5m", tolerance: 0 },
      }),
    }))
  })

  it("restores an edited subscription to global inheritance", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const fetchMock = installSubscriptionAPI([{
      id: "sub-1",
      name: "自定义订阅",
      url: "https://example.com/sub",
      interval_min: 60,
      last_updated: "2026-01-01T00:00:00Z",
      outbounds: [],
      urltest: { enabled: false, interval: "5m", tolerance: 0 },
    }])
    const user = userEvent.setup()
    renderApp(<App />, "/subscriptions")

    expect(await screen.findByText("URLTest：已关闭")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "编辑" }))
    expect(screen.getByRole("button", { name: "关闭" })).toHaveAttribute("data-pressed")
    expect(screen.getByLabelText("URLTest 测试间隔")).toHaveValue("5m")
    expect(screen.getByLabelText("URLTest 切换容差（毫秒）")).toHaveValue(0)
    await user.click(screen.getByRole("button", { name: "恢复全部默认值" }))
    await user.click(screen.getByRole("button", { name: "保存" }))

    expect(fetchMock).toHaveBeenCalledWith("/api/subscriptions/sub-1", expect.objectContaining({
      method: "PUT",
      body: JSON.stringify({
        name: "自定义订阅",
        url: "https://example.com/sub",
        interval_min: 60,
        urltest: null,
      }),
    }))
  })
})
