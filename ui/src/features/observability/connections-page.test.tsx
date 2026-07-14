import { screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

describe("ConnectionsPage", () => {
  it("shows live connections", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const encoder = new TextEncoder()
    const body = new ReadableStream({ start(controller) { controller.enqueue(encoder.encode(`data: ${JSON.stringify({ active_connections: 1, list: [{ id: 1, target: "example.com:443", outbound: "proxy", upload: 10, download: 20, start: new Date(Date.now() - 1000).toISOString() }] })}\n\n`)); controller.close() } })
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(body)))
    renderApp(<App />, "/observability/connections")

    expect(await screen.findByText("example.com:443")).toBeInTheDocument()
    expect(screen.getByText("1s")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "关闭全部连接" })).toBeInTheDocument()
  })
})
