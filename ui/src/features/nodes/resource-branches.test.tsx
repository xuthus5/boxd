import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

describe("resource empty and error states", () => {
  it("shows a node list load error", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify({ code: "internal_error", message: "nodes unavailable" }), { status: 500 }))))
    renderApp(<App />, "/nodes")
    expect(await screen.findByText("nodes unavailable", {}, { timeout: 3000 })).toBeInTheDocument()
  })

  it("shows an empty node state", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("[]")))
    renderApp(<App />, "/nodes")
    expect(await screen.findByText("暂无节点")).toBeInTheDocument()
  })

  it("shows subscription refresh errors", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify([{
      id: "sub", name: "失败订阅", url: "https://example.com", interval_min: 60,
      last_updated: "2026-01-01T00:00:00Z", error: "refresh failed",
    }]))))
    renderApp(<App />, "/subscriptions")
    expect(await screen.findByText("refresh failed")).toBeInTheDocument()
  })

  it("shows a failed node speed test", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      const data = path === "/api/nodes/"
        ? [{ tag: "bad", type: "vless", server: "bad.example", port: 443, raw: {} }]
        : path === "/api/nodes/groups" ? { groups: [] }
          : path === "/api/nodes/test-results" ? {}
            : { tag: "bad", test_type: "http", success: false, error: "timeout" }
      return Promise.resolve(new Response(JSON.stringify(data)))
    }))
    const user = userEvent.setup()
    renderApp(<App />, "/nodes")
    await user.click(await screen.findByRole("button", { name: "测速" }))
    expect(await screen.findByText("timeout")).toBeInTheDocument()
  })
})
