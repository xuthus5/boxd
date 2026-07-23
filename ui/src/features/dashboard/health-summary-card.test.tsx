import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { screen, waitFor } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { HealthSummaryCard } from "@/features/dashboard/health-summary-card"
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
})
