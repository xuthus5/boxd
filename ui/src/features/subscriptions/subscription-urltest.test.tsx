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
  it("locks URLTest fields while inheriting globals and only enables them after enabling", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const fetchMock = installSubscriptionAPI()
    const user = userEvent.setup()
    renderApp(<App />, "/subscriptions")

    await user.click(await screen.findByRole("button", { name: "新增订阅" }))
    const inheritURL = screen.getByRole("combobox", { name: "URLTest 测试地址" })
    expect(inheritURL).toBeDisabled()
    expect(inheritURL).toHaveTextContent(defaults.url)
    expect(screen.getByLabelText("URLTest 测试间隔")).toBeDisabled()
    expect(screen.getByLabelText("URLTest 测试间隔")).toHaveValue(defaults.interval)
    expect(screen.getByLabelText("URLTest 切换容差（毫秒）")).toBeDisabled()
    expect(screen.getByLabelText("URLTest 切换容差（毫秒）")).toHaveValue(defaults.tolerance)
    expect(screen.getAllByText("当前继承全局默认值，参数只读展示。").length).toBeGreaterThan(0)

    await user.type(screen.getByLabelText("名称"), "主订阅")
    await user.type(screen.getByLabelText("订阅 URL"), "https://example.com/sub")
    await user.click(screen.getByRole("button", { name: "启用" }))
    const enabledURL = screen.getByRole("combobox", { name: "URLTest 测试地址" })
    expect(enabledURL).toBeEnabled()
    expect(enabledURL).toHaveTextContent("留空继承全局")
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

  it("hides URLTest fields when disabled and explains selector fallback", async () => {
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
    expect(screen.queryByRole("combobox", { name: "URLTest 测试地址" })).not.toBeInTheDocument()
    expect(screen.queryByLabelText("URLTest 测试间隔")).not.toBeInTheDocument()
    expect(screen.getByText("已关闭 URLTest 组策略")).toBeInTheDocument()
    expect(screen.getByText(/该订阅组将使用 Selector 策略/)).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "保存" }))

    expect(fetchMock).toHaveBeenCalledWith("/api/subscriptions/sub-1", expect.objectContaining({
      method: "PUT",
      body: JSON.stringify({
        name: "自定义订阅",
        url: "https://example.com/sub",
        interval_min: 60,
        urltest: { enabled: false },
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
      urltest: { enabled: true, interval: "5m", tolerance: 0 },
    }])
    const user = userEvent.setup()
    renderApp(<App />, "/subscriptions")

    await user.click(await screen.findByRole("button", { name: "编辑" }))
    expect(screen.getByRole("button", { name: "启用" })).toHaveAttribute("data-pressed")
    expect(screen.getByLabelText("URLTest 测试间隔")).toHaveValue("5m")
    await user.click(screen.getByRole("button", { name: "恢复全部默认值" }))
    expect(screen.getByRole("button", { name: "继承全局" })).toHaveAttribute("data-pressed")
    expect(screen.getByLabelText("URLTest 测试间隔")).toBeDisabled()
    expect(screen.getByLabelText("URLTest 测试间隔")).toHaveValue(defaults.interval)
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
