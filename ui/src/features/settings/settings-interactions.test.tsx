import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { installMockAPI } from "@/test/mock-api"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear(); localStorage.clear() })

describe("settings interactions", () => {
  it("updates appearance and runtime preferences", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    installMockAPI()
    const user = userEvent.setup()
    renderApp(<App />, "/settings")
    await screen.findByRole("heading", { name: "应用设置" })
    await user.click(screen.getByText("深色"))
    await user.click(screen.getByText("深色"))
    await user.clear(screen.getByLabelText("测速地址"))
    await user.type(screen.getByLabelText("测速地址"), "https://example.com/test")
    await user.click(screen.getByRole("button", { name: "保存测速地址" }))
    await user.click(screen.getByRole("switch", { name: "内核自启" }))
    await user.click(screen.getByText("English"))
    await user.click(screen.getByText("English"))
    expect(screen.getByRole("heading", { name: "Application Settings" })).toBeInTheDocument()
    expect(localStorage.getItem("boxui.preferences.v1")).toContain("dark")
  })

  it("submits password and JWT rotations", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    installMockAPI()
    const user = userEvent.setup()
    renderApp(<App />, "/settings")
    await screen.findByRole("heading", { name: "应用设置" })
    await user.type(screen.getByLabelText("当前密码"), "current")
    await user.type(screen.getByLabelText("新密码"), "new-password")
    await user.click(screen.getByRole("button", { name: "轮换密码" }))
  })

  it("submits a JWT secret rotation", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    installMockAPI()
    const user = userEvent.setup()
    renderApp(<App />, "/settings")
    await screen.findByRole("heading", { name: "应用设置" })
    await user.type(screen.getByLabelText("JWT 签名密钥"), "replacement-secret")
    await user.click(screen.getByRole("button", { name: "轮换 JWT 密钥" }))
    expect(screen.getByRole("alertdialog")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "确认轮换" }))
  })

  it("saves subscription URLTest defaults and synchronizes configuration", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const fetchMock = installMockAPI()
    const user = userEvent.setup()
    renderApp(<App />, "/settings")
    await screen.findByText("订阅 URLTest 默认值")

    await user.clear(screen.getByLabelText("URLTest 测试地址"))
    await user.type(screen.getByLabelText("URLTest 测试地址"), "https://example.com/generate_204")
    await user.clear(screen.getByLabelText("URLTest 测试间隔"))
    await user.type(screen.getByLabelText("URLTest 测试间隔"), "5m")
    await user.clear(screen.getByLabelText("URLTest 切换容差（毫秒）"))
    await user.type(screen.getByLabelText("URLTest 切换容差（毫秒）"), "100")
    await user.click(screen.getByRole("button", { name: "保存 URLTest 默认值" }))

    expect(fetchMock).toHaveBeenCalledWith("/api/settings/urltest-defaults", expect.objectContaining({
      method: "PUT",
      body: JSON.stringify({
        enabled: true,
        url: "https://example.com/generate_204",
        interval: "5m",
        tolerance: 100,
      }),
    }))
    expect(fetchMock).toHaveBeenCalledWith("/api/nodes/sync-config", expect.objectContaining({ method: "POST" }))
  })

  it("rejects an invalid URLTest interval before saving", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    installMockAPI()
    const user = userEvent.setup()
    renderApp(<App />, "/settings")
    await screen.findByText("订阅 URLTest 默认值")

    await user.clear(screen.getByLabelText("URLTest 测试间隔"))
    await user.type(screen.getByLabelText("URLTest 测试间隔"), "0s")

    expect(screen.getByText("请输入大于 0 的时长，例如 3m 或 30s。")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "保存 URLTest 默认值" })).toBeDisabled()
  })
})
