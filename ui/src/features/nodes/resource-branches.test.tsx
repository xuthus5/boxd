import { screen, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

describe("node resource states", () => {
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
    expect(await screen.findAllByText("暂无节点")).toHaveLength(3)
  })
})

describe("subscription resource states", () => {
  it("shows subscription refresh errors", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      const data = path.endsWith("/urltest-defaults")
        ? { enabled: true, url: "https://www.gstatic.com/generate_204", interval: "3m", tolerance: 50 }
        : path.endsWith("/nodes/") ? [] : [{
          id: "sub", name: "失败订阅", url: "https://example.com", interval_min: 60,
          last_updated: "2026-01-01T00:00:00Z", error: "refresh failed",
        }]
      return Promise.resolve(new Response(JSON.stringify(data)))
    }))
    renderApp(<App />, "/subscriptions")
    expect(await screen.findByText("refresh failed")).toBeInTheDocument()
  })

  it("shows subscription list load errors", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() => Promise.resolve(new Response(
      JSON.stringify({ code: "internal_error", message: "subscriptions unavailable" }),
      { status: 500 },
    ))))
    renderApp(<App />, "/subscriptions")
    expect(await screen.findByText("subscriptions unavailable", {}, { timeout: 3000 })).toBeInTheDocument()
  })

  it("shows imported-node load errors on the subscriptions page", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path === "/api/subscriptions/") return Promise.resolve(new Response("[]"))
      if (path === "/api/nodes/") {
        return Promise.resolve(new Response(
          JSON.stringify({ code: "internal_error", message: "imported nodes unavailable" }),
          { status: 500 },
        ))
      }
      return Promise.resolve(new Response("{}"))
    }))
    renderApp(<App />, "/subscriptions")
    expect(await screen.findByText("imported nodes unavailable", {}, { timeout: 3000 })).toBeInTheDocument()
  })
})

describe("node test result states", () => {
  it("shows a failed node speed test", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    let tested = false
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request, init?: RequestInit) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path === "/api/nodes/test" && init?.method === "POST") tested = true
      const data = path === "/api/nodes/"
        ? [{ tag: "bad", type: "vless", server: "bad.example", port: 443, source: "import", raw: {} }]
        : path === "/api/nodes/groups" ? { groups: [] }
          : path === "/api/nodes/test-results" && tested
            ? { bad: { http: { tag: "bad", test_type: "http", success: false, error: "timeout" } } }
            : path === "/api/nodes/test-results" ? {}
            : { tag: "bad", test_type: "http", success: false, error: "timeout" }
      return Promise.resolve(new Response(JSON.stringify(data)))
    }))
    const user = userEvent.setup()
    renderApp(<App />, "/nodes")
    const all = await screen.findByRole("region", { name: "所有节点" })
    await user.click(within(all).getByRole("button", { name: "测速" }))
    await user.click(await screen.findByRole("menuitem", { name: "HTTP" }))
    expect(await screen.findAllByText("timeout")).toHaveLength(2)
  })
})
