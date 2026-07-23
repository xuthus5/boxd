import { screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => {
  vi.unstubAllGlobals()
  sessionStore.clear()
})

describe("advanced configuration", () => {
  it("opens the full configuration editor", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const raw = typeof input === "string"
        ? input
        : input instanceof URL
          ? input.pathname
          : new URL(input.url).pathname
      const path = raw.split("?")[0]
      if (path === "/api/settings/preferences") {
        return Promise.resolve(new Response(JSON.stringify({ theme: "system", language: "zh", minimumLogLevel: "all" })))
      }
      if (path === "/api/settings/password") {
        return Promise.resolve(new Response(JSON.stringify({ defaultPassword: false })))
      }
      if (path === "/api/service/status") {
        return Promise.resolve(new Response(JSON.stringify({ running: false })))
      }
      if (path.startsWith("/api/config")) {
        return Promise.resolve(new Response(JSON.stringify({ log: { level: "info" } })))
      }
      return Promise.resolve(new Response(JSON.stringify({})))
    }))
    renderApp(<App />, "/advanced/raw")
    expect(await screen.findByRole("heading", { name: "完整配置" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "保存完整配置" })).toBeInTheDocument()
    expect(screen.getByTestId("config-diff-panel")).toHaveTextContent("配置无变化")
  })
})
