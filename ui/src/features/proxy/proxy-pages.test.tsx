import { screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

describe("proxy configuration pages", () => {
  it("renders each inbound configuration as a card", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
      inbounds: [
        { tag: "mixed-in", type: "mixed", listen: "::", listen_port: 1080 },
        { tag: "tun-in", type: "tun", interface_name: "tun0" },
      ],
      outbounds: [],
    }))))

    renderApp(<App />, "/proxy/inbounds")

    expect(await screen.findByText("mixed-in")).toBeInTheDocument()
    expect(screen.getByText("tun-in")).toBeInTheDocument()
    expect(screen.getAllByRole("article")).toHaveLength(2)
    expect(screen.queryByRole("table")).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: "新增入站" })).toBeInTheDocument()
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
