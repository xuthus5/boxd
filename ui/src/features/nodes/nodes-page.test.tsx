import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { NodesPage } from "@/features/nodes/nodes-page"
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


describe("NodesPage stability filter", () => {
  it("filters nodes by stability and clears filters", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path === "/api/nodes/") {
        return Promise.resolve(new Response(JSON.stringify([
          { tag: "hk-stable", type: "vless", server: "a.example.com", port: 443, source: "import" },
          { tag: "us-unstable", type: "trojan", server: "b.example.com", port: 443, source: "import" },
        ])))
      }
      if (path === "/api/nodes/test-history") {
        return Promise.resolve(new Response(JSON.stringify({
          history: {
            "hk-stable": {
              tcp: [
                { timestamp: "1", success: true, latency_ms: 20 },
                { timestamp: "2", success: true, latency_ms: 22 },
                { timestamp: "3", success: true, latency_ms: 18 },
                { timestamp: "4", success: true, latency_ms: 19 },
              ],
            },
            "us-unstable": {
              tcp: [
                { timestamp: "1", success: false },
                { timestamp: "2", success: false },
                { timestamp: "3", success: true, latency_ms: 200 },
                { timestamp: "4", success: false },
              ],
            },
          },
        })))
      }
      if (path === "/api/nodes/groups") return Promise.resolve(new Response(JSON.stringify({ groups: [] })))
      if (path === "/api/nodes/test-results") return Promise.resolve(new Response(JSON.stringify({})))
      if (path === "/api/settings/preferences") {
        return Promise.resolve(new Response(JSON.stringify({ theme: "system", language: "zh", minimumLogLevel: "all" })))
      }
      if (path === "/api/settings/password") {
        return Promise.resolve(new Response(JSON.stringify({ defaultPassword: false })))
      }
      return Promise.resolve(new Response(JSON.stringify({})))
    }))
    const user = userEvent.setup()
    renderApp(<App />, "/nodes")

    const all = await screen.findByRole("region", { name: "所有节点" })
    expect(within(all).getByText("hk-stable")).toBeInTheDocument()
    expect(within(all).getByText("us-unstable")).toBeInTheDocument()

    await user.click(screen.getByRole("combobox", { name: "稳定性" }))
    await user.click(await screen.findByRole("option", { name: "稳定" }))
    expect(within(all).getByText("hk-stable")).toBeInTheDocument()
    expect(within(all).queryByText("us-unstable")).not.toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "清除筛选" }))
    expect(within(all).getByText("hk-stable")).toBeInTheDocument()
    expect(within(all).getByText("us-unstable")).toBeInTheDocument()
  })

  it("clears filters from the empty-state action", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path === "/api/nodes/") {
        return Promise.resolve(new Response(JSON.stringify([
          { tag: "hk-stable", type: "vless", server: "a.example.com", port: 443, source: "import" },
          { tag: "us-unstable", type: "trojan", server: "b.example.com", port: 443, source: "import" },
        ])))
      }
      if (path === "/api/nodes/test-history") {
        return Promise.resolve(new Response(JSON.stringify({
          history: {
            "hk-stable": {
              tcp: [
                { timestamp: "1", success: true, latency_ms: 20 },
                { timestamp: "2", success: true, latency_ms: 22 },
                { timestamp: "3", success: true, latency_ms: 18 },
                { timestamp: "4", success: true, latency_ms: 19 },
              ],
            },
            "us-unstable": {
              tcp: [
                { timestamp: "1", success: false },
                { timestamp: "2", success: false },
                { timestamp: "3", success: true, latency_ms: 200 },
                { timestamp: "4", success: false },
              ],
            },
          },
        })))
      }
      if (path === "/api/nodes/groups") return Promise.resolve(new Response(JSON.stringify({ groups: [] })))
      if (path === "/api/nodes/test-results") return Promise.resolve(new Response(JSON.stringify({})))
      if (path === "/api/settings/preferences") {
        return Promise.resolve(new Response(JSON.stringify({ theme: "system", language: "zh", minimumLogLevel: "all" })))
      }
      if (path === "/api/settings/password") {
        return Promise.resolve(new Response(JSON.stringify({ defaultPassword: false })))
      }
      return Promise.resolve(new Response(JSON.stringify({})))
    }))
    const user = userEvent.setup()
    renderApp(<App />, "/nodes?q=missing-node")

    expect(await screen.findByText("无匹配节点")).toBeInTheDocument()
    const clearButtons = screen.getAllByRole("button", { name: "清除筛选" })
    await user.click(clearButtons[clearButtons.length - 1])
    const all = await screen.findByRole("region", { name: "所有节点" })
    expect(within(all).getByText("hk-stable")).toBeInTheDocument()
    expect(within(all).getByText("us-unstable")).toBeInTheDocument()
  })

})


describe("NodesPage deep links", () => {
  it("seeds filters from deep-link query params", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path === "/api/nodes/") {
        return Promise.resolve(new Response(JSON.stringify([
          { tag: "hk-stable", type: "vless", server: "a.example.com", port: 443, source: "import" },
          { tag: "us-unstable", type: "trojan", server: "b.example.com", port: 443, source: "import" },
        ])))
      }
      if (path === "/api/nodes/test-history") {
        return Promise.resolve(new Response(JSON.stringify({
          history: {
            "hk-stable": {
              tcp: [
                { timestamp: "1", success: true, latency_ms: 20 },
                { timestamp: "2", success: true, latency_ms: 22 },
                { timestamp: "3", success: true, latency_ms: 18 },
                { timestamp: "4", success: true, latency_ms: 19 },
              ],
            },
            "us-unstable": {
              tcp: [
                { timestamp: "1", success: false },
                { timestamp: "2", success: false },
                { timestamp: "3", success: true, latency_ms: 200 },
                { timestamp: "4", success: false },
              ],
            },
          },
        })))
      }
      if (path === "/api/nodes/groups") return Promise.resolve(new Response(JSON.stringify({ groups: [] })))
      if (path === "/api/nodes/test-results") return Promise.resolve(new Response(JSON.stringify({})))
      if (path === "/api/settings/preferences") {
        return Promise.resolve(new Response(JSON.stringify({ theme: "system", language: "zh", minimumLogLevel: "all" })))
      }
      if (path === "/api/settings/password") {
        return Promise.resolve(new Response(JSON.stringify({ defaultPassword: false })))
      }
      return Promise.resolve(new Response(JSON.stringify({})))
    }))
    renderApp(<App />, "/nodes?q=hk&stability=stable&sort=latency")
    expect(await screen.findByLabelText("搜索节点")).toHaveValue("hk")
    expect(screen.getByRole("combobox", { name: "稳定性" })).toHaveTextContent("稳定")
    expect(screen.getByRole("combobox", { name: "排序节点" })).toHaveTextContent("按延迟")
    const all = await screen.findByRole("region", { name: "所有节点" })
    expect(within(all).getByText("hk-stable")).toBeInTheDocument()
    expect(within(all).queryByText("us-unstable")).not.toBeInTheDocument()
  })

  it("switches sort to stability after bulk speed test", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const fetchMock = vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      const body = path === "/api/nodes/" ? [
        { tag: "hk-1", type: "vless", server: "hk.example", port: 443, source: "subscription", source_name: "主订阅" },
      ] : path === "/api/nodes/test-batch" ? {
        results: [
          { tag: "hk-1", test_type: "tcp", success: true, latency_ms: 18 },
          { tag: "hk-1", test_type: "http", success: true, latency_ms: 22 },
          { tag: "hk-1", test_type: "icmp", success: false, error: "timeout" },
        ],
      } : path === "/api/nodes/groups" ? { groups: [] } : {}
      return Promise.resolve(new Response(JSON.stringify(body)))
    })
    vi.stubGlobal("fetch", fetchMock)
    const user = userEvent.setup()
    renderApp(<App />, "/nodes")
    const all = await screen.findByRole("region", { name: "所有节点" })
    expect(screen.getByRole("combobox", { name: "排序节点" })).toHaveTextContent("按名称")
    await user.click(within(all).getByRole("button", { name: "批量测速" }))
    await waitFor(() => {
      expect(screen.getByRole("combobox", { name: "排序节点" })).toHaveTextContent("按稳定性")
    })
    expect(await screen.findByText(/2\/3 成功/)).toBeInTheDocument()
  })
})

describe("NodesPage load densify", () => {
  it("densifies page load failure with retry", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response(JSON.stringify({
      status: "error", data: null, error: { code: "unavailable", message: "nodes unavailable" }, meta: null,
    }), { status: 503 }))))
    renderApp(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <NodesPage />
      </QueryClientProvider>,
    )
    const alert = await screen.findByTestId("page-load-error")
    expect(alert).toHaveAttribute("data-error-code", "unavailable")
    expect(screen.getByText("nodes unavailable")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "重试" })).toBeInTheDocument()
  })
})

describe("NodesPage partial query densify", () => {
  it("densifies history-only failures without blanking the page", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string"
        ? input
        : input instanceof URL
          ? input.pathname
          : "url" in input
            ? new URL(input.url).pathname
            : String(input)
      if (path.includes("/api/settings/preferences")) {
        return Promise.resolve(new Response(JSON.stringify({
          theme: "system", language: "zh", minimumLogLevel: "all",
        })))
      }
      if (path.includes("/api/settings/password")) {
        return Promise.resolve(new Response(JSON.stringify({ defaultPassword: false })))
      }
      if (path.includes("/api/nodes/test-history") || path.includes("/api/nodes/history")) {
        return Promise.resolve(new Response(JSON.stringify({
          status: "error", data: null, error: { code: "unavailable", message: "history unavailable" }, meta: null,
        }), { status: 503 }))
      }
      if (path.includes("/api/nodes/results")) {
        return Promise.resolve(new Response(JSON.stringify({})))
      }
      if (path.includes("/api/nodes/groups")) {
        return Promise.resolve(new Response(JSON.stringify({ groups: [] })))
      }
      if (path.includes("/api/nodes")) {
        return Promise.resolve(new Response(JSON.stringify([
          { tag: "hk-01", type: "vless", server: "example.com", port: 443, source: "import" },
        ])))
      }
      return Promise.resolve(new Response(JSON.stringify({})))
    }))
    renderApp(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <NodesPage />
      </QueryClientProvider>,
    )
    expect((await screen.findAllByText("hk-01")).length).toBeGreaterThan(0)
    const alert = await screen.findByTestId("card-query-error")
    expect(alert).toHaveAttribute("data-error-code", "unavailable")
    expect(screen.getByText("history unavailable")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "重试" })).toBeInTheDocument()
  })
})
