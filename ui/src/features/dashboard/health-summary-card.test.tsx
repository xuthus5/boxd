import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { screen, waitFor } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { HealthSummaryCard } from "@/features/dashboard/health-summary-card"
import type { ConnectionWithRates } from "@/features/observability/connection-rate"
import { renderApp } from "@/test/render"

afterEach(() => {
  vi.unstubAllGlobals()
  vi.clearAllMocks()
})

function renderCard() {
  return renderApp(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <HealthSummaryCard
        status={{ running: true, uptime: "1m" }}
        snapshot={{ active_connections: 0, list: [] }}
      />
    </QueryClientProvider>,
  )
}

describe("HealthSummaryCard", () => {
  it("shows the live rate and busiest outbound deep links", () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response(JSON.stringify({})))))
    const connection: ConnectionWithRates = {
      id: 1,
      target: "api.example.com:443",
      outbound: "proxy",
      network: "tcp",
      upload: 100,
      download: 200,
      uploadRate: 1024,
      downloadRate: 2048,
      start: "2026-07-26T00:00:00Z",
    }
    renderApp(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <HealthSummaryCard
          status={{ running: true, uptime: "1m" }}
          snapshot={{ active_connections: 1, list: [connection] }}
        />
      </QueryClientProvider>,
    )

    expect(screen.getByText(/实时速率:/)).toHaveTextContent("↑ 1.00 KB/s · ↓ 2.00 KB/s")
    expect(screen.getByRole("link", { name: "↑ 1.00 KB/s · ↓ 2.00 KB/s" })).toHaveAttribute(
      "href",
      "/observability/connections?sort=rate",
    )
    expect(screen.getAllByRole("link", { name: "proxy" }).some((link) => (
      link.getAttribute("href") === "/observability/connections?outbound=proxy&sort=rate"
    ))).toBe(true)
  })

  it("embeds failed subscription preview below ops chips", async () => {
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path.includes("/api/subscriptions")) {
        return Promise.resolve(new Response(JSON.stringify([
          {
            id: "bad",
            name: "失败订阅",
            url: "https://example.com/bad",
            interval_min: 60,
            last_updated: "2026-01-01T00:00:00Z",
            outbounds: [],
            error: "timeout",
            error_code: "timeout",
          },
        ])))
      }
      if (path.includes("/api/nodes/test-history") || path.includes("/api/nodes/history")) {
        return Promise.resolve(new Response(JSON.stringify({ history: {} })))
      }
      if (path.includes("/api/nodes")) {
        return Promise.resolve(new Response(JSON.stringify([])))
      }
      if (path.includes("/api/config/apply-history")) {
        return Promise.resolve(new Response(JSON.stringify({ events: [] })))
      }
      return Promise.resolve(new Response("{}"))
    }))
    renderCard()
    expect(await screen.findByText("1 个失败订阅")).toBeInTheDocument()
    await waitFor(() => {
      expect(document.querySelector('[data-slot="failed-subscriptions-preview"]')).not.toBeNull()
    })
    expect(screen.getByText("失败订阅")).toBeInTheDocument()
    expect(screen.getAllByText("timeout").length).toBeGreaterThan(0)
    expect(screen.getByRole("link", { name: "查看: 失败订阅" })).toHaveAttribute(
      "href",
      "/subscriptions?q=%E5%A4%B1%E8%B4%A5%E8%AE%A2%E9%98%85&status=error",
    )
  })

  it("embeds problem node preview for failed and unstable nodes", async () => {
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path.includes("/api/subscriptions")) {
        return Promise.resolve(new Response(JSON.stringify([])))
      }
      if (path.includes("/api/nodes/test-history")) {
        return Promise.resolve(new Response(JSON.stringify({
          history: {
            "hk-bad": {
              tcp: [
                { timestamp: "2026-07-23T00:00:00Z", success: false },
                { timestamp: "2026-07-23T00:01:00Z", success: false },
              ],
            },
          },
        })))
      }
      if (path.includes("/api/nodes")) {
        return Promise.resolve(new Response(JSON.stringify([
          { tag: "hk-bad", type: "vless", server: "a.example.com", port: 443, source: "import" },
        ])))
      }
      if (path.includes("/api/config/apply-history")) {
        return Promise.resolve(new Response(JSON.stringify({ events: [] })))
      }
      return Promise.resolve(new Response("{}"))
    }))
    renderCard()
    expect(await screen.findByText("1 个全失败节点")).toBeInTheDocument()
    await waitFor(() => {
      expect(document.querySelector('[data-slot="problem-nodes-preview"]')).not.toBeNull()
    })
    expect(screen.getByText("hk-bad")).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "查看: hk-bad" })).toHaveAttribute(
      "href",
      "/nodes?q=hk-bad&stability=failed",
    )
  })

  it("shows densified stream error with copy control", async () => {
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path.includes("/api/config/apply-history")) {
        return Promise.resolve(new Response(JSON.stringify({ events: [] })))
      }
      if (path.includes("/api/subscriptions") || path.includes("/api/nodes")) {
        return Promise.resolve(new Response(JSON.stringify(path.includes("history") ? { history: {} } : [])))
      }
      return Promise.resolve(new Response("{}"))
    }))
    renderApp(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <HealthSummaryCard
          status={{ running: true, uptime: "1m" }}
          snapshot={{ active_connections: 0, list: [] }}
          streamError="failed to fetch connections"
          streamStatus="error"
        />
      </QueryClientProvider>,
    )
    expect(await screen.findByText("failed to fetch connections")).toBeInTheDocument()
    expect(document.querySelector('[data-slot="health-stream-error"]')).not.toBeNull()
    expect(screen.getByRole("button", { name: "复制流错误" })).toBeInTheDocument()
  })

  it("shows apply failure preview with path jump", async () => {
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path.includes("/api/config/apply-history")) {
        return Promise.resolve(new Response(JSON.stringify({
          events: [{
            id: "1",
            source: "validate_inbounds",
            status: "validate_failed",
            hash: "deadbeef",
            size: 12,
            error: "inbounds[0].listen_port: invalid",
            error_code: "config_invalid",
            applied_at: "2026-07-24T12:00:00Z",
          }],
        })))
      }
      if (path.includes("/api/subscriptions") || path.includes("/api/nodes")) {
        return Promise.resolve(new Response(JSON.stringify(path.includes("history") ? { history: {} } : [])))
      }
      return Promise.resolve(new Response("{}"))
    }))
    renderCard()
    await waitFor(() => {
      expect(document.querySelector('[data-slot="apply-failure-preview"]')).not.toBeNull()
    })
    const preview = document.querySelector('[data-slot="apply-failure-preview"]')
    expect(preview).not.toBeNull()
    expect(preview?.textContent).toMatch(/1 次配置应用\/校验失败/)
    expect(screen.getByText("inbounds[0].listen_port")).toBeInTheDocument()
    const openLinks = screen.getAllByRole("link", { name: "打开失败来源" })
    expect(openLinks.length).toBeGreaterThan(0)
    for (const link of openLinks) {
      expect(link).toHaveAttribute("href", "/advanced/raw?path=inbounds%5B0%5D.listen_port")
    }
    expect(preview?.querySelector('button[aria-label="复制错误"]')).not.toBeNull()
  })

  it("densifies health ops query failure with retry", async () => {
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path.includes("/api/subscriptions")) {
        return Promise.resolve(new Response(JSON.stringify({ error: "boom" }), { status: 503 }))
      }
      if (path.includes("/api/nodes/test-history") || path.includes("/api/nodes/history")) {
        return Promise.resolve(new Response(JSON.stringify({ history: {} })))
      }
      if (path.includes("/api/nodes")) {
        return Promise.resolve(new Response(JSON.stringify([])))
      }
      if (path.includes("/api/config/apply-history")) {
        return Promise.resolve(new Response(JSON.stringify({ events: [] })))
      }
      return Promise.resolve(new Response("{}"))
    }))
    renderCard()
    expect(await screen.findByTestId("card-query-error")).toBeInTheDocument()
    expect(document.querySelector('[data-error-code]')).not.toBeNull()
    expect(screen.getByRole("button", { name: "重试" })).toBeInTheDocument()
  })
})
