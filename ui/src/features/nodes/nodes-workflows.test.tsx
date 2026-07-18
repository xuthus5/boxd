import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

function setup(handler: (path: string, init?: RequestInit) => unknown, route = "/nodes") {
  sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
  const fetchMock = vi.fn((input: string | URL | Request, init?: RequestInit) => {
    const path = typeof input === "string" ? input : input.toString()
    return Promise.resolve(new Response(JSON.stringify(handler(path, init))))
  })
  vi.stubGlobal("fetch", fetchMock)
  renderApp(<App />, route)
  return { fetchMock, user: userEvent.setup() }
}

describe("node management workflows", () => {
  it("edits an imported node while preserving its advanced configuration", async () => {
    const { fetchMock, user } = setup((path) => {
      if (path === "/api/nodes/") return [{ tag: "hk-01", type: "vless", server: "example.com", port: 443, source: "import" }]
      if (path === "/api/nodes/hk-01") return { tag: "hk-01", type: "vless", server: "example.com", port: 443, raw: { uuid: "secret", tls: { enabled: true } } }
      if (path === "/api/subscriptions/") return []
      if (path === "/api/nodes/groups") return { groups: [] }
      return {}
    }, "/subscriptions")

    const card = await screen.findByRole("article", { name: "hk-01" })
    await user.click(within(card).getByRole("button", { name: "编辑" }))
    await user.clear(await screen.findByLabelText("服务器"))
    await user.type(screen.getByLabelText("服务器"), "new.example.com")
    await user.click(screen.getByRole("button", { name: "保存" }))

    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(
      "/api/nodes/hk-01",
      expect.objectContaining({
        method: "PUT",
        body: JSON.stringify({
          tag: "hk-01",
          type: "vless",
          server: "new.example.com",
          port: 443,
          config: { uuid: "secret", tls: { enabled: true } },
        }),
      }),
    ))
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/nodes/sync-config",
      expect.objectContaining({ method: "POST" }),
    )
  })

  it("switches test type and displays persisted results", async () => {
    const { fetchMock, user } = setup((path) => {
      if (path === "/api/nodes/") return [{ tag: "hk-01", type: "vless", server: "example.com", port: 443, source: "import" }]
      if (path === "/api/nodes/test-results") return { "hk-01_tcp": { tcp: { tag: "hk-01", test_type: "tcp", success: true, latency_ms: 18 } } }
      if (path === "/api/nodes/groups") return { groups: [] }
      if (path === "/api/nodes/test") return { tag: "hk-01", test_type: "tcp", success: true, latency_ms: 18 }
      return {}
    })

    expect(await screen.findAllByText("18 ms")).toHaveLength(2)
    const all = screen.getByRole("region", { name: "所有节点" })
    await user.click(within(all).getByRole("button", { name: "测速" }))
    await user.click(await screen.findByRole("menuitem", { name: "TCP" }))

    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(
      "/api/nodes/test",
      expect.objectContaining({ body: expect.stringContaining('"test_type":"tcp"') }),
    ))
  })
})

describe("node page responsibilities", () => {
  it("does not render runtime group controls", async () => {
    setup((path) => {
      if (path === "/api/nodes/") return []
      if (path === "/api/nodes/groups") return { groups: [{ type: "selector", tag: "proxy", now: "a", all: ["a", "b"] }] }
      return {}
    })

    await screen.findByRole("heading", { name: "节点" })
    expect(screen.queryByText("proxy")).not.toBeInTheDocument()
    expect(screen.queryByRole("combobox", { name: "proxy" })).not.toBeInTheDocument()
  })
})
