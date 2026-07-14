import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

function sse(data: unknown) {
  const encoder = new TextEncoder()
  return new Response(new ReadableStream({ start(controller) { controller.enqueue(encoder.encode(`data: ${JSON.stringify(data)}\n\n`)); controller.close() } }))
}

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

describe("LogsPage", () => {
  it("shows stream connection errors", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(null, { status: 503 })))
    renderApp(<App />, "/observability/logs")
    expect(await screen.findAllByText("SSE request failed with status 503")).toHaveLength(2)
  })

  it("shows log source tabs inside the log page", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(sse({ level: "info", message: "kernel ready", timestamp: "2026-01-01T00:00:00Z" })))
    renderApp(<App />, "/observability/logs")

    expect(await screen.findByRole("tab", { name: "内核日志" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "应用日志" })).toBeInTheDocument()
    expect(await screen.findByText("kernel ready")).toBeInTheDocument()
  })

  it("shows error logs without a timestamp", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(sse({ level: "error", message: "failed", timestamp: "" })))
    renderApp(<App />, "/observability/logs")
    expect(await screen.findByText("failed")).toBeInTheDocument()
    expect(screen.queryByText("时间")).not.toBeInTheDocument()
  })

  it("preserves each tab filter while switching sources", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() => Promise.resolve(sse({ level: "info", message: "ready" }))))
    const user = userEvent.setup()
    renderApp(<App />, "/observability/logs")
    const filter = (await screen.findAllByLabelText("搜索日志"))[0]
    await user.type(filter, "kernel")
    await user.click(screen.getByRole("tab", { name: "应用日志" }))
    await user.click(screen.getByRole("tab", { name: "内核日志" }))
    expect(filter).toHaveValue("kernel")
  })
})
