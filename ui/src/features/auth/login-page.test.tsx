import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { useAuth } from "@/features/auth/auth-context"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => {
  vi.unstubAllGlobals()
  sessionStore.clear()
})

describe("login page", () => {
  it("rejects auth hook usage outside its provider", () => {
    function Consumer() { useAuth(); return null }
    expect(() => renderApp(<Consumer />)).toThrow("useAuth must be used inside AuthProvider")
  })

  it("validates required credentials", async () => {
    const user = userEvent.setup()
    renderApp(<App />, "/login")
    await user.click(screen.getByRole("button", { name: "登录" }))
    expect(await screen.findByText("请输入用户名")).toBeInTheDocument()
    expect(screen.getByText("请输入密码")).toBeInTheDocument()
  })

  it("stores the session and opens the dashboard", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
      token: "token",
      expires_at: "2099-01-01T00:00:00Z",
    }))))
    const user = userEvent.setup()
    renderApp(<App />, "/login")

    await user.type(screen.getByLabelText("用户名"), "admin")
    await user.type(screen.getByLabelText("密码"), "secret")
    await user.click(screen.getByRole("button", { name: "登录" }))

    expect(await screen.findByRole("heading", { name: "仪表盘" })).toBeInTheDocument()
    expect(sessionStore.get()?.token).toBe("token")
  })

  it("shows the backend login error", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
      status: "error",
      data: null,
      error: { code: "unauthorized", message: "invalid credentials" },
      meta: null,
    }), { status: 401 })))
    const user = userEvent.setup()
    renderApp(<App />, "/login")

    await user.type(screen.getByLabelText("用户名"), "admin")
    await user.type(screen.getByLabelText("密码"), "wrong")
    await user.click(screen.getByRole("button", { name: "登录" }))

    expect(await screen.findByText("invalid credentials")).toBeInTheDocument()
  })
})
