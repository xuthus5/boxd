import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

function setup(handler: (path: string, init?: RequestInit) => unknown) {
  sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
  const fetchMock = vi.fn((input: string | URL | Request, init?: RequestInit) => {
    const path = typeof input === "string" ? input : input.toString()
    return Promise.resolve(new Response(JSON.stringify(handler(path, init))))
  })
  vi.stubGlobal("fetch", fetchMock)
  renderApp(<App />, "/nodes")
  return { fetchMock, user: userEvent.setup() }
}

describe("node management workflows", () => {
  it("edits an imported node while preserving its advanced configuration", async () => {
    const { fetchMock, user } = setup((path) => {
      if (path === "/api/nodes/") return [{ tag: "hk-01", type: "vless", server: "example.com", port: 443, source: "import" }]
      if (path === "/api/nodes/hk-01") return { tag: "hk-01", type: "vless", server: "example.com", port: 443, raw: { uuid: "secret", tls: { enabled: true } } }
      if (path === "/api/nodes/groups") return { groups: [] }
      return {}
    })

    await user.click(await screen.findByRole("button", { name: "编辑" }))
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
  })

  it("switches test type and displays persisted results", async () => {
    const { fetchMock, user } = setup((path) => {
      if (path === "/api/nodes/") return [{ tag: "hk-01", type: "vless", server: "example.com", port: 443, source: "import" }]
      if (path === "/api/nodes/test-results") return { "hk-01": { tcp: { tag: "hk-01", test_type: "tcp", success: true, latency_ms: 18 } } }
      if (path === "/api/nodes/groups") return { groups: [] }
      if (path === "/api/nodes/test") return { tag: "hk-01", test_type: "tcp", success: true, latency_ms: 18 }
      return {}
    })

    expect(await screen.findByText("18 ms")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "TCP" }))
    await user.click(screen.getByRole("button", { name: "测速" }))

    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(
      "/api/nodes/test",
      expect.objectContaining({ body: expect.stringContaining('"test_type":"tcp"') }),
    ))
  })
})

describe("node runtime groups", () => {
  it("selects selector members and runs URLTest groups", async () => {
    const { fetchMock, user } = setup((path) => {
      if (path === "/api/nodes/") return []
      if (path === "/api/nodes/groups") return { groups: [
        { type: "selector", tag: "proxy", now: "a", all: ["a", "b"] },
        { type: "urltest", tag: "auto", now: "a", all: ["a", "b"] },
      ] }
      if (path === "/api/nodes/groups/auto/urltest") return { a: 12, b: 24 }
      return {}
    })

    expect(await screen.findByText("proxy")).toBeInTheDocument()
    await user.click(screen.getByRole("combobox", { name: "proxy" }))
    await user.click(screen.getByRole("option", { name: "b" }))
    await user.click(screen.getByRole("button", { name: "运行 auto URLTest" }))

    expect(await screen.findByText("a: 12 ms")).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/nodes/selectors/proxy/select",
      expect.objectContaining({ method: "POST", body: JSON.stringify({ tag: "b" }) }),
    )
  })
})
