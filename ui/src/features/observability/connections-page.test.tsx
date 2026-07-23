import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

describe("ConnectionsPage", () => {
  it("shows live connections with rule and supports filtering", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const encoder = new TextEncoder()
    const body = new ReadableStream({
      start(controller) {
        controller.enqueue(encoder.encode(`data: ${JSON.stringify({
          active_connections: 2,
          list: [
            {
              id: 1,
              target: "example.com:443",
              outbound: "proxy",
              rule: "geosite-google",
              upload: 10,
              download: 20,
              start: new Date(Date.now() - 1000).toISOString(),
            },
            {
              id: 2,
              target: "cdn.example.net:443",
              outbound: "direct",
              rule: "geoip-cn",
              upload: 1,
              download: 2,
              start: new Date(Date.now() - 2000).toISOString(),
            },
          ],
        })}\n\n`))
        controller.close()
      },
    })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const raw = typeof input === "string"
        ? input
        : input instanceof URL
          ? input.pathname
          : new URL(input.url).pathname
      const path = raw.split("?")[0]
      if (path === "/api/stats/connections") {
        return Promise.resolve(new Response(body, { headers: { "Content-Type": "text/event-stream" } }))
      }
      if (path === "/api/settings/preferences") {
        return Promise.resolve(new Response(JSON.stringify({
          theme: "system", language: "zh", minimumLogLevel: "all",
        })))
      }
      if (path === "/api/settings/password") {
        return Promise.resolve(new Response(JSON.stringify({ defaultPassword: false })))
      }
      return Promise.resolve(new Response(JSON.stringify({})))
    }))
    const user = userEvent.setup()
    renderApp(<App />, "/observability/connections")

    expect(await screen.findByText("example.com:443")).toBeInTheDocument()
    expect(screen.getByText("显示 2 条")).toBeInTheDocument()
    expect(screen.getByText(/2 个出口/)).toBeInTheDocument()
    expect(screen.getByText("geosite-google")).toBeInTheDocument()
    expect(screen.getByText("1s")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "关闭全部连接" })).toBeInTheDocument()

    await user.type(screen.getByLabelText("搜索连接"), "direct")
    expect(screen.queryByText("example.com:443")).not.toBeInTheDocument()
    expect(screen.getByText("cdn.example.net:443")).toBeInTheDocument()
  })
})
