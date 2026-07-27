import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"
import { MemoryRouter } from "react-router-dom"
import { toast } from "sonner"

import { NodeCard } from "@/features/nodes/node-card"
import { i18n } from "@/i18n"
import * as copy from "@/lib/clipboard"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

function renderCard(results: Record<string, unknown>) {
  return render(
    <I18nextProvider i18n={i18n}>
      <MemoryRouter>
        <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
          <NodeCard
            node={{ tag: "hk", type: "vless", server: "hk.example", port: 443, source: "import" }}
            results={results as never}
          />
        </QueryClientProvider>
      </MemoryRouter>
    </I18nextProvider>,
  )
}

afterEach(() => {
  vi.restoreAllMocks()
  vi.clearAllMocks()
})

describe("NodeCard latency tone", () => {
  it("renders colored latency badges for success results", () => {
    renderCard({
      tcp: { tag: "hk", test_type: "tcp", success: true, latency_ms: 40 },
      http: { tag: "hk", test_type: "http", success: true, latency_ms: 220 },
      icmp: { tag: "hk", test_type: "icmp", success: false, error: "timeout" },
    })
    expect(screen.getByText("40 ms")).toBeInTheDocument()
    expect(screen.getByText("220 ms")).toBeInTheDocument()
    expect(screen.getByText("timeout")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "复制测速错误: ICMP timeout" })).toBeInTheDocument()
  })

  it("copies failed probe diagnostics from the error badge", async () => {
    const spy = vi.spyOn(copy, "copyText").mockResolvedValue()
    renderCard({
      icmp: {
        tag: "hk",
        test_type: "icmp",
        success: false,
        error: "timeout",
        timestamp: "2026-07-24T00:00:00Z",
      },
    })
    await userEvent.setup().click(screen.getByRole("button", { name: "复制测速错误: ICMP timeout" }))
    await waitFor(() => expect(spy).toHaveBeenCalled())
    expect(spy.mock.calls[0][0]).toContain("error: timeout")
    expect(spy.mock.calls[0][0]).toContain("tag: hk")
    expect(toast.success).toHaveBeenCalledWith("测速错误已复制")
  })
})
