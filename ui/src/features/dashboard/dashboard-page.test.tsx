import { screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"
import { formatBytes } from "@/features/dashboard/format"

function responseFor(path: string) {
  if (path === "/api/service/status") return { running: true, uptime: "1m", version: "1.13" }
  if (path === "/api/stats/traffic/history") return { points: [{ upload_bytes: 10, download_bytes: 20, timestamp: "2026-01-01T00:00:00Z" }] }
  if (path === "/api/runtime/memory") return { alloc: 1024, total: 2048, sys: 4096, num_gc: 2, heap_inuse: 512, stack_inuse: 128 }
  if (path === "/api/runtime/version") return { version: "dev", kernel_version: "1.13.14" }
  return null
}

function eventStream(data: unknown) {
  return new Response(`data: ${JSON.stringify(data)}\n\n`, {
    headers: { "Content-Type": "text/event-stream" },
  })
}

afterEach(() => {
  vi.unstubAllGlobals()
  sessionStore.clear()
})

describe("DashboardPage", () => {
  it("formats larger byte values", () => {
    expect(formatBytes(1024 ** 3)).toBe("1.00 GB")
  })
  it("shows service, traffic, memory, and version data", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path === "/api/stats/traffic") return Promise.resolve(eventStream({ upload_bytes: 30, download_bytes: 40, timestamp: "2026-01-01T00:00:01Z" }))
      if (path === "/api/stats/logs") return Promise.resolve(eventStream({ level: "info", message: "ready" }))
      return Promise.resolve(new Response(JSON.stringify(responseFor(path))))
    }))

    renderApp(<App />, "/dashboard")

    expect(await screen.findByText("运行中")).toBeInTheDocument()
    expect(screen.getByText("1.00 KB")).toBeInTheDocument()
    expect(screen.getByText("1.13.14")).toBeInTheDocument()
    expect(await screen.findByText(/下载 40 B/)).toBeInTheDocument()
    expect(screen.getByText("ready")).toBeInTheDocument()
  })
})
