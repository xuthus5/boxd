import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

describe("proxy configuration pages", () => {
  it("renders each inbound configuration as a card", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response(JSON.stringify({
      inbounds: [
        { tag: "mixed-in", type: "mixed", listen: "::", listen_port: 1080 },
        { tag: "tun-in", type: "tun", interface_name: "tun0" },
      ],
      outbounds: [],
    })))))

    renderApp(<App />, "/proxy/inbounds")

    expect(await screen.findByText("mixed-in")).toBeInTheDocument()
    expect(screen.getByText("tun-in")).toBeInTheDocument()
    expect(screen.getAllByRole("article")).toHaveLength(2)
    expect(screen.queryByRole("table")).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: "新增入站" })).toBeInTheDocument()
    expect(screen.getByLabelText("搜索配置")).toBeInTheDocument()
  })

  it("renders each outbound configuration as a card", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = String(input)
      const data = path === "/api/subscriptions/" ? [] : path === "/api/nodes/groups" ? { groups: [] }
        : { inbounds: [], outbounds: [{ tag: "proxy", type: "vless", server: "example.com", server_port: 443 }] }
      return Promise.resolve(new Response(JSON.stringify(data)))
    }))
    renderApp(<App />, "/proxy/outbounds")
    expect(await screen.findByText("proxy")).toBeInTheDocument()
    expect(screen.getByText("example.com:443")).toBeInTheDocument()
    expect(screen.getByRole("article")).toBeInTheDocument()
    expect(screen.queryByRole("table")).not.toBeInTheDocument()
  })

  it("groups subscription nodes behind their selector card", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = String(input)
      const data = path === "/api/subscriptions/" ? [{ id: "sub", name: "主订阅", url: "https://example.com", interval_min: 60, last_updated: "", outbounds: [{ tag: "hk", type: "vless" }, { tag: "us", type: "trojan" }] }]
        : path === "/api/nodes/groups" ? { groups: [{ type: "selector", tag: "主订阅", now: "hk", all: ["hk", "us"] }] }
        : { outbounds: [{ type: "vless", tag: "hk", server: "hk.example", server_port: 443 }, { type: "trojan", tag: "us", server: "us.example", server_port: 443 }, { type: "selector", tag: "主订阅", outbounds: ["hk", "us"] }, { type: "direct", tag: "direct" }] }
      return Promise.resolve(new Response(JSON.stringify(data)))
    }))
    renderApp(<App />, "/proxy/outbounds")
    expect(await screen.findByRole("combobox", { name: "主订阅" })).toHaveTextContent("hk")
    expect(screen.queryByText("hk.example:443")).not.toBeInTheDocument()
    expect(screen.queryByText("us.example:443")).not.toBeInTheDocument()
    expect(screen.getByRole("heading", { name: "direct" })).toBeInTheDocument()
  })

  it("uses the subscription URLTest group when runtime data is not yet available", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = String(input)
      const data = path === "/api/subscriptions/" ? [{ id: "sub", name: "自动订阅", url: "https://example.com", interval_min: 60, last_updated: "", outbounds: [{ tag: "node", type: "vless" }] }]
        : path === "/api/nodes/groups" ? { groups: [] }
        : { outbounds: [{ type: "vless", tag: "node", server: "node.example", server_port: 443 }, { type: "urltest", tag: "自动订阅", outbounds: ["node"] }] }
      return Promise.resolve(new Response(JSON.stringify(data)))
    }))
    renderApp(<App />, "/proxy/outbounds")
    expect(await screen.findByRole("button", { name: "运行 自动订阅 URLTest" })).toBeInTheDocument()
    expect(screen.queryByText("node.example:443")).not.toBeInTheDocument()
  })

  it("shows config and runtime group type mismatch on outbound cards", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = String(input)
      const data = path === "/api/subscriptions/"
        ? [{ id: "1", name: "自动订阅", url: "https://example.com", interval_min: 60, last_updated: "", outbounds: [{ tag: "node", type: "vless" }] }]
        : path === "/api/nodes/groups"
          ? { groups: [{ type: "selector", tag: "自动订阅", now: "node", all: ["node"] }] }
          : { outbounds: [{ type: "urltest", tag: "自动订阅", outbounds: ["node"], url: "https://cp.cloudflare.com/", interval: "3m", tolerance: 50 }] }
      return Promise.resolve(new Response(JSON.stringify(data)))
    }))
    renderApp(<App />, "/proxy/outbounds")
    expect(await screen.findByText("配置与运行时不一致")).toBeInTheDocument()
    expect(screen.getByText("配置与运行时类型不一致")).toBeInTheDocument()
  })
})

describe("proxy list deep links", () => {
  it("seeds inbound search from deep-link query params", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response(JSON.stringify({
      inbounds: [
        { tag: "mixed-in", type: "mixed", listen: "::", listen_port: 1080 },
        { tag: "tun-in", type: "tun", interface_name: "tun0" },
      ],
      outbounds: [],
    })))))
    renderApp(<App />, "/proxy/inbounds?q=tun")
    expect(await screen.findByLabelText("搜索配置")).toHaveValue("tun")
    expect(screen.getByText("tun-in")).toBeInTheDocument()
    expect(screen.queryByText("mixed-in")).not.toBeInTheDocument()
  })

  it("filters outbounds from search input and clears via URL state", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = String(input)
      const data = path === "/api/subscriptions/" ? [] : path === "/api/nodes/groups" ? { groups: [] }
        : {
          inbounds: [],
          outbounds: [
            { tag: "proxy", type: "vless", server: "example.com", server_port: 443 },
            { tag: "direct", type: "direct" },
          ],
        }
      return Promise.resolve(new Response(JSON.stringify(data)))
    }))
    const user = userEvent.setup()
    renderApp(<App />, "/proxy/outbounds")
    expect(await screen.findByText("proxy")).toBeInTheDocument()
    await user.type(screen.getByLabelText("搜索配置"), "direct")
    expect(screen.getAllByText("direct").length).toBeGreaterThan(0)
    expect(screen.queryByText("proxy")).not.toBeInTheDocument()
    expect(screen.queryByText("example.com:443")).not.toBeInTheDocument()
  })
})


describe("proxy path jump", () => {
  it("jumps save error path into inbound editor advanced JSON", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const fetchMock = vi.fn((input: string | URL | Request, init?: RequestInit) => {
      const path = String(typeof input === "string" ? input : input instanceof URL ? input.pathname : input.url).split("?")[0]
      if (path === "/api/config/" && init?.method === "PUT") {
        return Promise.resolve(new Response(JSON.stringify({
          status: "error",
          data: null,
          error: { code: "config_invalid_runtime", message: "inbounds[0].listen_port: invalid" },
          meta: {},
        }), { status: 500 }))
      }
      if (path === "/api/config/" || path === "/api/config/raw") {
        return Promise.resolve(new Response(JSON.stringify({
          inbounds: [{ tag: "mixed-in", type: "mixed", listen: "::", listen_port: 1080 }],
          outbounds: [],
        })))
      }
      return Promise.resolve(new Response(JSON.stringify({})))
    })
    vi.stubGlobal("fetch", fetchMock)
    const user = userEvent.setup()
    renderApp(<App />, "/proxy/inbounds")
    await screen.findByText("mixed-in")
    await user.click(screen.getByRole("button", { name: "删除" }))
    await user.click(screen.getByRole("button", { name: "确认删除" }))
    expect(await screen.findByTestId("config-save-error")).toBeInTheDocument()
    expect(screen.getAllByText(/inbounds\[0\]\.listen_port/).length).toBeGreaterThan(0)
    await user.click(screen.getByRole("button", { name: "跳转到路径" }))
    expect(await screen.findByRole("dialog")).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "高级 JSON" })).toHaveAttribute("aria-selected", "true")
  })

  it("opens outbound editor advanced JSON from path deep link", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = String(input)
      if (path.startsWith("/api/subscriptions/") || path.includes("/api/nodes/groups")) {
        return Promise.resolve(new Response(JSON.stringify(path.includes("groups") ? { groups: [] } : [])))
      }
      return Promise.resolve(new Response(JSON.stringify({
        inbounds: [],
        outbounds: [{ tag: "proxy", type: "vless", server: "example.com", server_port: 443 }],
      })))
    }))
    renderApp(<App />, "/proxy/outbounds?path=outbounds%5B0%5D.server")
    expect(await screen.findByRole("dialog")).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "高级 JSON" })).toHaveAttribute("aria-selected", "true")
  })
})


describe("proxy dry-run validate", () => {
  it("validates inbound draft without writing", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const fetchMock = vi.fn((input: string | URL | Request, init?: RequestInit) => {
      const path = String(typeof input === "string" ? input : input instanceof URL ? input.pathname : input.url).split("?")[0]
      if (path === "/api/config/validate" && init?.method === "POST") {
        return Promise.resolve(new Response(JSON.stringify({
          status: "ok", data: { valid: true }, error: null, meta: { validated: true, applied: false },
        })))
      }
      if (path === "/api/config/" || path === "/api/config/raw") {
        return Promise.resolve(new Response(JSON.stringify({
          inbounds: [{ tag: "mixed-in", type: "mixed", listen: "::", listen_port: 1080 }],
          outbounds: [],
        })))
      }
      return Promise.resolve(new Response(JSON.stringify({})))
    })
    vi.stubGlobal("fetch", fetchMock)
    const user = userEvent.setup()
    renderApp(<App />, "/proxy/inbounds")
    await screen.findByText("mixed-in")
    await user.click(screen.getByRole("button", { name: "编辑" }))
    expect(await screen.findByRole("dialog")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "校验配置" }))
    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith("/api/config/validate", expect.objectContaining({ method: "POST" }))
    })
    expect(fetchMock.mock.calls.some((call) => String(call[0]) === "/api/config/" && call[1]?.method === "PUT")).toBe(false)
  })
})
