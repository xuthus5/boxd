import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"
import { toast } from "sonner"

import { NodeResultsCard } from "@/features/nodes/node-results-card"
import { i18n } from "@/i18n"
import * as copy from "@/lib/clipboard"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
  vi.clearAllMocks()
})

function renderCard() {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
    bad: {
      http: {
        tag: "bad",
        test_type: "http",
        success: false,
        error: "timeout",
        timestamp: "2026-07-24T00:00:00Z",
      },
    },
    ok: {
      tcp: {
        tag: "ok",
        test_type: "tcp",
        success: true,
        latency_ms: 42,
      },
    },
  }))))
  return render(
    <I18nextProvider i18n={i18n}>
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <NodeResultsCard visibleTags={new Set(["bad", "ok"])} />
      </QueryClientProvider>
    </I18nextProvider>,
  )
}

describe("NodeResultsCard", () => {
  it("copies failed probe diagnostics from the status badge", async () => {
    const spy = vi.spyOn(copy, "copyText").mockResolvedValue()
    renderCard()
    const button = await screen.findByRole("button", { name: "复制测速错误: bad HTTP timeout" })
    await userEvent.setup().click(button)
    await waitFor(() => expect(spy).toHaveBeenCalled())
    expect(spy.mock.calls[0][0]).toContain("error: timeout")
    expect(spy.mock.calls[0][0]).toContain("tag: bad")
    expect(toast.success).toHaveBeenCalledWith("测速错误已复制")
    expect(screen.getByText("42 ms")).toBeInTheDocument()
  })
})
