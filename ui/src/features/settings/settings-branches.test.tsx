import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

describe("settings alternate states", () => {
  it("shows a load error instead of dereferencing failed query data", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify({ code: "internal_error", message: "settings unavailable" }), { status: 500 }))))
    renderApp(<App />, "/settings")
    expect(await screen.findByText("settings unavailable", {}, { timeout: 3000 })).toBeInTheDocument()
  })

  it("warns when the default password is active", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      const data = path.endsWith("/password") ? { defaultPassword: true }
        : path.endsWith("/jwt-secret") ? { masked: "", present: false, length: 0 }
          : path.endsWith("/urltest-defaults") ? { enabled: true, url: "https://example.com/generate_204", interval: "3m", tolerance: 50 }
          : path.endsWith("/url-test") ? { url: "" } : { enabled: false }
      return Promise.resolve(new Response(JSON.stringify(data)))
    }))
    renderApp(<App />, "/settings")
    expect(await screen.findByText("默认密码仍在使用")).toBeInTheDocument()
  })
})

describe("settings save failures", () => {
  it("reports a test URL save error", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request, init?: RequestInit) => {
      const path = typeof input === "string" ? input : input.toString()
      if (init?.method === "PUT") return Promise.resolve(new Response(JSON.stringify({ code: "internal_error", message: "save failed" }), { status: 500 }))
      const data = path.endsWith("/password") ? { defaultPassword: false }
        : path.endsWith("/jwt-secret") ? { masked: "x", present: true, length: 1 }
          : path.endsWith("/urltest-defaults") ? { enabled: true, url: "https://example.com/generate_204", interval: "3m", tolerance: 50 }
          : path.endsWith("/url-test") ? { url: "https://example.com" } : { enabled: false }
      return Promise.resolve(new Response(JSON.stringify(data)))
    }))
    const user = userEvent.setup()
    renderApp(<App />, "/settings")
    await user.click(await screen.findByRole("button", { name: "保存测速地址" }))
    expect(await screen.findByText("save failed")).toBeInTheDocument()
  })

  it("restores kernel autostart when saving fails", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request, init?: RequestInit) => {
      const path = typeof input === "string" ? input : input.toString()
      if (init?.method === "PUT") return Promise.resolve(new Response(JSON.stringify({ code: "internal_error", message: "save failed" }), { status: 500 }))
      const data = path.endsWith("/password") ? { defaultPassword: false }
        : path.endsWith("/jwt-secret") ? { masked: "x", present: true, length: 1 }
          : path.endsWith("/urltest-defaults") ? { enabled: true, url: "https://example.com/generate_204", interval: "3m", tolerance: 50 }
          : path.endsWith("/url-test") ? { url: "https://example.com" } : { enabled: false }
      return Promise.resolve(new Response(JSON.stringify(data)))
    }))
    const user = userEvent.setup()
    renderApp(<App />, "/settings")
    const autostart = await screen.findByRole("switch", { name: "内核自启" })
    await user.click(autostart)
    expect(await screen.findByText("save failed")).toBeInTheDocument()
    expect(autostart).not.toBeChecked()
  })
})

describe("settings credential failures", () => {
  it("reports password and JWT rotation errors", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request, init?: RequestInit) => {
      const path = typeof input === "string" ? input : input.toString()
      if (init?.method === "PUT") return Promise.resolve(new Response(JSON.stringify({ code: "invalid_request", message: "rotation failed" }), { status: 400 }))
      const data = path.endsWith("/password") ? { defaultPassword: false }
        : path.endsWith("/jwt-secret") ? { masked: "x", present: true, length: 1 }
          : path.endsWith("/urltest-defaults") ? { enabled: true, url: "https://example.com/generate_204", interval: "3m", tolerance: 50 }
          : path.endsWith("/url-test") ? { url: "https://example.com" } : { enabled: false }
      return Promise.resolve(new Response(JSON.stringify(data)))
    }))
    const user = userEvent.setup()
    renderApp(<App />, "/settings")
    await screen.findByRole("heading", { name: "应用设置" })
    await user.type(screen.getByLabelText("当前密码"), "current")
    await user.type(screen.getByLabelText("新密码"), "new-password")
    await user.click(screen.getByRole("button", { name: "轮换密码" }))
    expect(await screen.findByText("rotation failed")).toBeInTheDocument()
    await user.type(screen.getByLabelText("JWT 签名密钥"), "replacement")
    await user.click(screen.getByRole("button", { name: "轮换 JWT 密钥" }))
    await user.click(screen.getByRole("button", { name: "确认轮换" }))
  })
})
