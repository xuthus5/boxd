import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

describe("NodesPage", () => {
  it("lists nodes without import or manual sync actions", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      const body = path === "/api/nodes/" ? [{ tag: "hk-01", type: "vless", server: "example.com", port: 443, raw: {} }] : {}
      return Promise.resolve(new Response(JSON.stringify(body)))
    }))
    const { container } = renderApp(<App />, "/nodes")

    expect(await screen.findByText("hk-01")).toBeInTheDocument()
    expect(container.querySelector("[data-slot=card] [data-slot=card]")).not.toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "导入节点" })).not.toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "同步到配置" })).not.toBeInTheDocument()
  })

  it("does not offer deletion for subscription nodes", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      const body = path === "/api/nodes/"
        ? [{ tag: "sub-node", type: "vless", source: "subscription", source_name: "主订阅" }]
        : { groups: [] }
      return Promise.resolve(new Response(JSON.stringify(body)))
    }))

    renderApp(<App />, "/nodes")

    const all = await screen.findByRole("region", { name: "所有节点" })
    const card = within(all).getByRole("article", { name: "sub-node" })
    expect(within(card).queryByRole("button", { name: "编辑" })).not.toBeInTheDocument()
    expect(within(card).queryByRole("button", { name: "删除" })).not.toBeInTheDocument()
  })

  it("labels a subscription node when its source name is absent", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify([
      { tag: "sub-node", type: "vless", source: "subscription" },
    ]))))
    renderApp(<App />, "/nodes")
    expect(await screen.findByText("订阅")).toBeInTheDocument()
  })

})

describe("NodesPage single tests", () => {
  it("runs a selected speed test from a node card", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const fetchMock = vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      const body = path === "/api/nodes/"
        ? [{ tag: "hk-01", type: "vless", server: "example.com", port: 443, source: "import" }]
        : path === "/api/nodes/test" ? { tag: "hk-01", test_type: "tcp", success: true, latency_ms: 18 } : { groups: [] }
      return Promise.resolve(new Response(JSON.stringify(body)))
    })
    vi.stubGlobal("fetch", fetchMock)
    const user = userEvent.setup()
    renderApp(<App />, "/nodes")

    const all = await screen.findByRole("region", { name: "所有节点" })
    const card = within(all).getByRole("article", { name: "hk-01" })
    const trigger = within(card).getByRole("button", { name: "测速" })
    expect(trigger.closest("[data-slot=card-header]")).not.toBeNull()
    await user.click(trigger)
    await user.click(await screen.findByRole("menuitem", { name: "TCP" }))

    expect(fetchMock).toHaveBeenCalledWith("/api/nodes/test", expect.objectContaining({
      method: "POST",
      body: expect.stringContaining('"test_type":"tcp"'),
    }))
  })

  it("disables every speed-test action while a request is pending", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    let finishTest: (response: Response) => void = () => undefined
    const pendingTest = new Promise<Response>((resolve) => { finishTest = resolve })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path === "/api/nodes/") return Promise.resolve(new Response(JSON.stringify([
        { tag: "hk-01", type: "vless", server: "example.com", port: 443, source: "import" },
      ])))
      if (path === "/api/nodes/test") return pendingTest
      if (path === "/api/nodes/groups") return Promise.resolve(new Response(JSON.stringify({ groups: [] })))
      return Promise.resolve(new Response("{}"))
    }))
    const user = userEvent.setup()
    renderApp(<App />, "/nodes")

    const all = await screen.findByRole("region", { name: "所有节点" })
    const card = within(all).getByRole("article", { name: "hk-01" })
    const trigger = within(card).getByRole("button", { name: "测速" })
    await user.click(trigger)
    await user.click(await screen.findByRole("menuitem", { name: "TCP" }))
    expect(trigger).toBeDisabled()

    finishTest(new Response(JSON.stringify({ tag: "hk-01", test_type: "tcp", success: true, latency_ms: 18 })))
    await waitFor(() => expect(within(card).getByRole("button", { name: "测速" })).toBeEnabled())
  })
})

describe("NodesPage source regions", () => {
  it("shows all nodes and repeats them in source regions", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      const body = path === "/api/nodes/" ? [
        { tag: "manual-1", type: "vless", server: "manual.example", port: 443, source: "import" },
        { tag: "hk-1", type: "vless", server: "hk.example", port: 443, source: "subscription", source_name: "香港订阅" },
        { tag: "us-1", type: "trojan", server: "us.example", port: 443, source: "subscription", source_name: "美国订阅" },
      ] : path === "/api/nodes/groups" ? { groups: [] } : {}
      return Promise.resolve(new Response(JSON.stringify(body)))
    }))
    renderApp(<App />, "/nodes")

    const all = await screen.findByRole("region", { name: "所有节点" })
    const subscriptions = screen.getByRole("region", { name: "订阅节点" })
    const imported = screen.getByRole("region", { name: "手动导入节点" })
    expect(within(all).getByRole("article", { name: "manual-1" })).toBeInTheDocument()
    expect(within(all).getByRole("article", { name: "hk-1" })).toBeInTheDocument()
    expect(within(subscriptions).getByRole("heading", { name: "香港订阅" })).toBeInTheDocument()
    expect(within(subscriptions).getByRole("heading", { name: "美国订阅" })).toBeInTheDocument()
    expect(within(imported).getByRole("article", { name: "manual-1" })).toBeInTheDocument()
  })
})

describe("NodesPage batch tests", () => {
  it("runs all three test types from one node card", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const fetchMock = vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      const body = path === "/api/nodes/"
        ? [{ tag: "hk-1", type: "vless", server: "hk.example", port: 443, source: "subscription", source_name: "香港订阅" }]
        : path === "/api/nodes/test-batch" ? { results: [] } : path === "/api/nodes/groups" ? { groups: [] } : {}
      return Promise.resolve(new Response(JSON.stringify(body)))
    })
    vi.stubGlobal("fetch", fetchMock)
    const user = userEvent.setup()
    renderApp(<App />, "/nodes")

    const all = await screen.findByRole("region", { name: "所有节点" })
    const card = within(all).getByRole("article", { name: "hk-1" })
    await user.click(within(card).getByRole("button", { name: "测速" }))
    for (const name of ["全部", "TCP", "HTTP", "ICMP"]) {
      expect(await screen.findByRole("menuitem", { name })).toBeInTheDocument()
    }
    await user.click(screen.getByRole("menuitem", { name: "全部" }))
    expect(fetchMock).toHaveBeenCalledWith("/api/nodes/test-batch", expect.objectContaining({
      method: "POST",
      body: JSON.stringify({ items: [
        { tag: "hk-1", test_type: "tcp", server: "hk.example", port: 443 },
        { tag: "hk-1", test_type: "http", server: "hk.example", port: 443 },
        { tag: "hk-1", test_type: "icmp", server: "hk.example", port: 443 },
      ], concurrency: 3 }),
    }))
  })
})

describe("NodesPage group batch tests", () => {
  it("runs every valid node in one card group", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const fetchMock = vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      const body = path === "/api/nodes/" ? [
        { tag: "hk-1", type: "vless", server: "hk.example", port: 443, source: "subscription", source_name: "主订阅" },
        { tag: "us-1", type: "trojan", server: "us.example", port: 8443, source: "subscription", source_name: "主订阅" },
        { tag: "invalid", type: "direct", source: "import" },
      ] : path === "/api/nodes/test-batch" ? { results: [] } : path === "/api/nodes/groups" ? { groups: [] } : {}
      return Promise.resolve(new Response(JSON.stringify(body)))
    })
    vi.stubGlobal("fetch", fetchMock)
    renderApp(<App />, "/nodes")

    const all = await screen.findByRole("region", { name: "所有节点" })
    await userEvent.setup().click(within(all).getByRole("button", { name: "批量测速" }))
    expect(fetchMock).toHaveBeenCalledWith("/api/nodes/test-batch", expect.objectContaining({
      method: "POST",
      body: JSON.stringify({ items: [
        { tag: "hk-1", test_type: "tcp", server: "hk.example", port: 443 },
        { tag: "hk-1", test_type: "http", server: "hk.example", port: 443 },
        { tag: "hk-1", test_type: "icmp", server: "hk.example", port: 443 },
        { tag: "us-1", test_type: "tcp", server: "us.example", port: 8443 },
        { tag: "us-1", test_type: "http", server: "us.example", port: 8443 },
        { tag: "us-1", test_type: "icmp", server: "us.example", port: 8443 },
      ], concurrency: 8 }),
    }))
  })
})
