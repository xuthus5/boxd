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
})
