import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => {
  vi.unstubAllGlobals()
  sessionStore.clear()
})

function mockFetch(handler: (path: string, init?: RequestInit) => Response | Promise<Response>) {
  vi.stubGlobal("fetch", vi.fn((input: string | URL | Request, init?: RequestInit) => {
    const raw = typeof input === "string"
      ? input
      : input instanceof URL
        ? input.pathname
        : new URL(input.url).pathname
    const path = raw.split("?")[0]
    return Promise.resolve(handler(path, init))
  }))
}

function json(data: unknown, status = 200) {
  return new Response(JSON.stringify(data), { status })
}

function basePath(path: string, init?: RequestInit): Response | null {
  if (path === "/api/settings/preferences") {
    return json({ theme: "system", language: "zh", minimumLogLevel: "all" })
  }
  if (path === "/api/settings/password") return json({ defaultPassword: false })
  if (path === "/api/service/status") return json({ running: false })
  if (init?.method === "PUT") return null
  return null
}

describe("advanced alternate states", () => {
  it("shows a raw configuration query error", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    mockFetch((path) => {
      const base = basePath(path)
      if (base) return base
      if (path.startsWith("/api/config")) return json({ code: "internal_error", message: "raw failed" }, 500)
      return json({})
    })
    renderApp(<App />, "/advanced/raw")
    expect(await screen.findByText("raw failed", {}, { timeout: 3000 })).toBeInTheDocument()
  })

  it("uses a default value when a section is absent", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    mockFetch((path) => {
      const base = basePath(path)
      if (base) return base
      if (path === "/api/config/" || path === "/api/config/raw") return json({})
      return json({})
    })
    renderApp(<App />, "/advanced/endpoints")
    expect(await screen.findByRole("button", { name: "保存配置" })).toBeEnabled()
  })

  it("reports raw configuration rollback", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    mockFetch((path, init) => {
      const base = basePath(path, init)
      if (base) return base
      if (init?.method === "PUT") return json({ status: "rolled_back", data: null, error: null, meta: {} })
      if (path.startsWith("/api/config")) return json({ log: {} })
      return json({})
    })
    const user = userEvent.setup()
    renderApp(<App />, "/advanced/raw")
    await user.click(await screen.findByRole("button", { name: "保存完整配置" }))
    await user.click(screen.getByRole("button", { name: "确认覆盖" }))
    expect(await screen.findByText("配置保存未生效，后端已回滚。")).toBeInTheDocument()
  })

  it("reports raw configuration save errors", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    mockFetch((path, init) => {
      const base = basePath(path, init)
      if (base) return base
      if (init?.method === "PUT") return json({ code: "internal_error", message: "write failed" }, 500)
      if (path.startsWith("/api/config")) return json({ log: {} })
      return json({})
    })
    const user = userEvent.setup()
    renderApp(<App />, "/advanced/raw")
    await user.click(await screen.findByRole("button", { name: "保存完整配置" }))
    await user.click(screen.getByRole("button", { name: "确认覆盖" }))
    expect(await screen.findByTestId("config-save-error", {}, { timeout: 3000 })).toHaveTextContent(/write failed/)
  })
})
