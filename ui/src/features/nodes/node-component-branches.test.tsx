import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

function authenticate() {
  sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
  return userEvent.setup()
}

describe("node component branches", () => {
  it("shows node editor load failures", async () => {
    const user = authenticate()
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path === "/api/nodes/") return Promise.resolve(new Response(JSON.stringify([{ tag: "bad", type: "vless", source: "import" }])))
      if (path === "/api/nodes/bad") return Promise.resolve(new Response(JSON.stringify({ code: "internal_error", message: "node load failed" }), { status: 500 }))
      if (path === "/api/nodes/groups") return Promise.resolve(new Response(JSON.stringify({ groups: [] })))
      return Promise.resolve(new Response("{}"))
    }))
    renderApp(<App />, "/nodes")
    await user.click(await screen.findByRole("button", { name: "编辑" }))
    expect(await screen.findByText("node load failed", {}, { timeout: 3000 })).toBeInTheDocument()
  })

  it("defaults optional node editor fields", async () => {
    const user = authenticate()
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path === "/api/nodes/") return Promise.resolve(new Response(JSON.stringify([{ tag: "minimal", type: "direct", source: "import" }])))
      if (path === "/api/nodes/minimal") return Promise.resolve(new Response(JSON.stringify({ tag: "minimal", type: "direct" })))
      if (path === "/api/nodes/groups") return Promise.resolve(new Response(JSON.stringify({ groups: [] })))
      return Promise.resolve(new Response("{}"))
    }))
    renderApp(<App />, "/nodes")
    await user.click(await screen.findByRole("button", { name: "编辑" }))
    expect(await screen.findByLabelText("服务器")).toHaveValue("")
    expect(screen.getByLabelText("端口")).toHaveValue(null)
    expect(screen.getByLabelText("节点高级 JSON")).toBeInTheDocument()
  })
})

describe("node result and runtime branches", () => {
  it("renders failed persisted results without latency", async () => {
    authenticate()
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path === "/api/nodes/") return Promise.resolve(new Response(JSON.stringify([
        { tag: "bad", type: "vless", server: "bad.example", port: 443, source: "import" },
      ])))
      if (path === "/api/nodes/groups") return Promise.resolve(new Response(JSON.stringify({ groups: [] })))
      return Promise.resolve(new Response(JSON.stringify({ bad: { http: { tag: "bad", test_type: "http", success: false, error: "timeout" } } })))
    }))
    renderApp(<App />, "/nodes")
    expect(await screen.findByText("timeout")).toBeInTheDocument()
    expect(screen.getByText("—")).toBeInTheDocument()
  })

  it("hides persisted results for nodes that no longer exist", async () => {
    authenticate()
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path === "/api/nodes/") return Promise.resolve(new Response(JSON.stringify([
        { tag: "current", type: "vless", server: "current.example", port: 443, source: "import" },
      ])))
      if (path === "/api/nodes/groups") return Promise.resolve(new Response(JSON.stringify({ groups: [] })))
      return Promise.resolve(new Response(JSON.stringify({
        current: { http: { tag: "current", test_type: "http", success: false, error: "current result" } },
        stale: { http: { tag: "stale", test_type: "http", success: false, error: "stale result" } },
      })))
    }))
    renderApp(<App />, "/nodes")
    await screen.findByText("current result")
    expect(screen.queryByText("stale result")).not.toBeInTheDocument()
  })

  it("reports URLTest failures", async () => {
    const user = authenticate()
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request, init?: RequestInit) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path === "/api/nodes/") return Promise.resolve(new Response("[]"))
      if (path === "/api/nodes/groups") return Promise.resolve(new Response(JSON.stringify({ groups: [{ type: "urltest", tag: "auto", now: "a", all: ["a"] }] })))
      if (init?.method === "POST") return Promise.resolve(new Response(JSON.stringify({ code: "unavailable", message: "kernel stopped" }), { status: 503 }))
      return Promise.resolve(new Response("{}"))
    }))
    renderApp(<App />, "/nodes")
    await user.click(await screen.findByRole("button", { name: "运行 auto URLTest" }))
    expect(await screen.findByText("kernel stopped")).toBeInTheDocument()
  })
})
