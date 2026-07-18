import { screen, within } from "@testing-library/react"
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
      if (path === "/api/subscriptions/") return Promise.resolve(new Response("[]"))
      return Promise.resolve(new Response("{}"))
    }))
    renderApp(<App />, "/subscriptions")
    const card = await screen.findByRole("article", { name: "bad" })
    await user.click(within(card).getByRole("button", { name: "编辑" }))
    expect(await screen.findByText("node load failed", {}, { timeout: 3000 })).toBeInTheDocument()
  })

  it("defaults optional node editor fields", async () => {
    const user = authenticate()
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path === "/api/nodes/") return Promise.resolve(new Response(JSON.stringify([{ tag: "minimal", type: "direct", source: "import" }])))
      if (path === "/api/nodes/minimal") return Promise.resolve(new Response(JSON.stringify({ tag: "minimal", type: "direct" })))
      if (path === "/api/subscriptions/") return Promise.resolve(new Response("[]"))
      return Promise.resolve(new Response("{}"))
    }))
    renderApp(<App />, "/subscriptions")
    const card = await screen.findByRole("article", { name: "minimal" })
    await user.click(within(card).getByRole("button", { name: "编辑" }))
    expect(await screen.findByLabelText("服务器")).toHaveValue("")
    expect(screen.getByLabelText("端口")).toHaveValue(null)
    const editor = screen.getByLabelText("节点高级 JSON")
    expect(editor).toBeInTheDocument()
    await user.clear(editor)
    await user.type(editor, "{{}")
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
  })
})

describe("node persisted result branches", () => {
  it("keeps sub-millisecond latency visible", async () => {
    authenticate()
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path === "/api/nodes/") return Promise.resolve(new Response(JSON.stringify([
        { tag: "fast", type: "vless", server: "fast.example", port: 443, source: "import" },
      ])))
      if (path === "/api/nodes/test-results") return Promise.resolve(new Response(JSON.stringify({
        fast_tcp: { tcp: { tag: "fast", test_type: "tcp", success: true, latency_ms: 0.3 } },
      })))
      return Promise.resolve(new Response(JSON.stringify({ groups: [] })))
    }))
    renderApp(<App />, "/nodes")
    expect(await screen.findAllByText("0.3 ms")).toHaveLength(2)
  })

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
    expect(await screen.findAllByText("timeout")).toHaveLength(2)
    expect(screen.getAllByText("—")).toHaveLength(4)
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
    await screen.findAllByText("current result")
    expect(screen.queryByText("stale result")).not.toBeInTheDocument()
  })
})

describe("node fallback and runtime branches", () => {
  it("shows result fallbacks and disables nodes without a test target", async () => {
    authenticate()
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path === "/api/nodes/") return Promise.resolve(new Response(JSON.stringify([
        { tag: "partial", type: "direct", source: "import" },
      ])))
      if (path === "/api/nodes/test-results") return Promise.resolve(new Response(JSON.stringify({
        partial: {
          tcp: { tag: "partial", test_type: "tcp", success: false },
          http: { tag: "partial", test_type: "http", success: true },
        },
      })))
      if (path === "/api/nodes/groups") return Promise.resolve(new Response(JSON.stringify({ groups: [] })))
      return Promise.resolve(new Response("{}"))
    }))
    renderApp(<App />, "/nodes")
    const all = await screen.findByRole("region", { name: "所有节点" })
    const card = within(all).getByRole("article", { name: "partial" })
    expect(within(all).getByRole("button", { name: "批量测速" })).toBeDisabled()
    expect(within(card).getByRole("button", { name: "测速" })).toBeDisabled()
    expect(within(card).getByText("测速失败")).toBeInTheDocument()
    expect(within(card).getByText("正常")).toBeInTheDocument()
    expect(within(card).getByText("—", { selector: "dd *" })).toBeInTheDocument()
  })

})
