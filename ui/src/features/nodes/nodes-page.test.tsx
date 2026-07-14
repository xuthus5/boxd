import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

describe("NodesPage", () => {
  it("lists nodes and exposes import and sync actions", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      const body = path === "/api/nodes/" ? [{ tag: "hk-01", type: "vless", server: "example.com", port: 443, raw: {} }] : {}
      return Promise.resolve(new Response(JSON.stringify(body)))
    }))
    renderApp(<App />, "/nodes")

    expect(await screen.findByText("hk-01")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "导入节点" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "同步到配置" })).toBeInTheDocument()
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

    expect(await screen.findByText("主订阅")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "订阅节点" })).toBeDisabled()
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

describe("NodesPage batch tests", () => {
  it("runs a batch speed test for listed nodes", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const fetchMock = vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      const body = path === "/api/nodes/"
        ? [{ tag: "hk-01", type: "vless", server: "example.com", port: 443, source: "import" }]
        : path === "/api/nodes/test-batch" ? { results: [] } : { groups: [] }
      return Promise.resolve(new Response(JSON.stringify(body)))
    })
    vi.stubGlobal("fetch", fetchMock)
    const user = userEvent.setup()
    renderApp(<App />, "/nodes")

    await user.click(await screen.findByRole("button", { name: "TCP" }))
    await user.click(await screen.findByRole("button", { name: "批量测速" }))

    expect(fetchMock).toHaveBeenCalledWith("/api/nodes/test-batch", expect.objectContaining({
      method: "POST",
      body: expect.stringContaining('"test_type":"tcp"'),
    }))
  })
})
