import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

describe("advanced alternate states", () => {
  it("shows a raw configuration query error", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify({
      code: "internal_error", message: "raw failed",
    }), { status: 500 }))))
    renderApp(<App />, "/advanced/raw")
    expect(await screen.findByText("raw failed", {}, { timeout: 3000 })).toBeInTheDocument()
  })

  it("uses a default value when a section is absent", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("{}")))
    renderApp(<App />, "/advanced/endpoints")
    expect(await screen.findByRole("button", { name: "保存配置" })).toBeEnabled()
  })

  it("reports raw configuration rollback", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((_input: string | URL | Request, init?: RequestInit) => {
      if (init?.method === "PUT") return Promise.resolve(new Response(JSON.stringify({ status: "rolled_back", data: null, error: null, meta: {} })))
      return Promise.resolve(new Response(JSON.stringify({ log: {} })))
    }))
    const user = userEvent.setup()
    renderApp(<App />, "/advanced/raw")
    await user.click(await screen.findByRole("button", { name: "保存完整配置" }))
    await user.click(screen.getByRole("button", { name: "确认覆盖" }))
    expect(await screen.findByText("配置保存未生效，后端已回滚。")).toBeInTheDocument()
  })

  it("reports raw configuration save errors", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((_input: string | URL | Request, init?: RequestInit) => {
      if (init?.method === "PUT") return Promise.resolve(new Response(JSON.stringify({ code: "internal_error", message: "write failed" }), { status: 500 }))
      return Promise.resolve(new Response(JSON.stringify({ log: {} })))
    }))
    const user = userEvent.setup()
    renderApp(<App />, "/advanced/raw")
    await user.click(await screen.findByRole("button", { name: "保存完整配置" }))
    await user.click(screen.getByRole("button", { name: "确认覆盖" }))
    expect(await screen.findByText("write failed")).toBeInTheDocument()
  })
})
