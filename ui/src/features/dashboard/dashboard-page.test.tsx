import { screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"
import { formatBytes } from "@/features/dashboard/format"

function responseFor(path: string) {
  if (path === "/api/service/status") return { running: true, uptime: "1m", version: "1.13" }
  if (path === "/api/stats/traffic/history") return { points: [{ upload_bytes: 10, download_bytes: 20, timestamp: "2026-01-01T00:00:00Z" }] }
  if (path === "/api/runtime/memory") return { alloc: 1024, total: 2048, sys: 4096, num_gc: 2, heap_inuse: 512, stack_inuse: 128, num_goroutine: 12 }
  if (path === "/api/runtime/version") return { version: "dev", kernel_version: "1.13.14" }
  if (path === "/api/nodes/groups") return { groups: [{ tag: "proxy", type: "selector", now: "a", all: ["a", "b"] }] }
  if (path === "/api/config/" || path === "/api/config") {
    return {
      inbounds: [{ tag: "mixed-in", type: "mixed", listen: "::", listen_port: 1080 }],
      outbounds: [{ tag: "proxy", type: "selector", outbounds: ["direct"] }],
      route: { rules: [{ outbound: "proxy" }] },
      experimental: { clash_api: { external_controller: "127.0.0.1:9090" } },
    }
  }
  if (path === "/api/subscriptions/" || path === "/api/subscriptions") {
    return [{ id: "sub-1", name: "主订阅", url: "https://example.com/sub", interval_min: 60, last_updated: "2026-01-01T00:00:00Z", outbounds: [] }]
  }
  if (path === "/api/runtime/clash-mode") return { mode: "Rule", mode_list: ["Rule", "Global", "Direct"] }
  if (path === "/api/config/apply-history") return { events: [] }
  return null
}

function eventStream(data: unknown) {
  return new Response(`data: ${JSON.stringify(data)}\n\n`, {
    headers: { "Content-Type": "text/event-stream" },
  })
}

function eventStreams(items: unknown[]) {
  return new Response(items.map((item) => `data: ${JSON.stringify(item)}\n\n`).join(""), {
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
      const path = typeof input === "string" ? input : input instanceof URL ? input.pathname : "url" in input ? new URL(input.url).pathname : String(input)
      if (path === "/api/stats/traffic") return Promise.resolve(eventStream({ upload_bytes: 30, download_bytes: 40, timestamp: "2026-01-01T00:00:01Z" }))
      if (path === "/api/stats/connections") return Promise.resolve(eventStream({ active_connections: 2, list: [{ id: 1, target: "example.com:443", outbound: "proxy", rule: "geosite-google", network: "tcp", upload: 10, download: 20, start: "2026-01-01T00:00:00Z" }, { id: 2, target: "cdn.example.net:443", outbound: "direct", rule: "geoip-cn", network: "udp", upload: 1, download: 2, start: "2026-01-01T00:00:01Z" }] }))
      if (path === "/api/stats/logs") return Promise.resolve(eventStream({ level: "info", message: "ready" }))
      return Promise.resolve(new Response(JSON.stringify(responseFor(path))))
    }))

    renderApp(<App />, "/dashboard")

    expect(await screen.findByText("运行中")).toBeInTheDocument()
    expect(await screen.findByText("运行健康")).toBeInTheDocument()
    expect(screen.getByText("2 条活跃连接")).toBeInTheDocument()
    expect(screen.getByText(/主要出口/)).toBeInTheDocument()
    expect(screen.getByText("运行时统计")).toBeInTheDocument()
    expect(screen.getByText("1.00 KB")).toBeInTheDocument()
    expect(screen.getByText("1.13.14")).toBeInTheDocument()
    expect(screen.getByText("运维入口")).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "错误日志" })).toHaveAttribute("href", "/observability/logs?preset=errors")
    expect(screen.getByRole("button", { name: "启动" })).toHaveClass("h-8")
    expect(screen.getByRole("button", { name: "停止" })).toHaveClass("h-8")
    expect(screen.getByRole("button", { name: "重启" })).toHaveClass("h-8")
    expect(screen.getByRole("link", { name: "错误日志" })).toHaveClass("h-8")
    expect(await screen.findByText(/下载 20 B\/s/)).toBeInTheDocument()
    expect(screen.getByText("ready")).toBeInTheDocument()
    // Setup checklist hides itself once all steps are complete and no subscription failures exist.
    expect(screen.queryByText("快速上手")).not.toBeInTheDocument()
  })

  it("keeps the latest twenty dashboard logs", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input instanceof URL ? input.pathname : "url" in input ? new URL(input.url).pathname : String(input)
      if (path === "/api/stats/traffic") return Promise.resolve(eventStream({ upload_bytes: 0, download_bytes: 0, timestamp: "2026-01-01T00:00:01Z" }))
      if (path === "/api/stats/connections") return Promise.resolve(eventStream({ active_connections: 0, list: [] }))
      if (path === "/api/stats/logs") return Promise.resolve(eventStreams(Array.from({ length: 25 }, (_, index) => ({ level: "info", message: `log-${index}` }))))
      return Promise.resolve(new Response(JSON.stringify(responseFor(path))))
    }))

    renderApp(<App />, "/dashboard")

    expect(await screen.findByText("log-24")).toBeInTheDocument()
    expect(screen.queryByText("log-4")).not.toBeInTheDocument()
    expect(screen.getAllByText(/^log-/)).toHaveLength(20)
  })
})
