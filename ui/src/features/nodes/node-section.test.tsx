import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"
import { MemoryRouter } from "react-router-dom"
import { toast } from "sonner"

import { NodeSection } from "@/features/nodes/node-section"
import { i18n } from "@/i18n"

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
}))

afterEach(() => {
  vi.unstubAllGlobals()
  vi.clearAllMocks()
})

function wrap() {
  return render(
    <I18nextProvider i18n={i18n}>
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })}>
        <MemoryRouter>
          <NodeSection
            title="测试组"
            description="2 个节点"
            nodes={[
              { tag: "hk-01", type: "vless", server: "a.example", port: 443, source: "import" },
              { tag: "us-01", type: "trojan", server: "b.example", port: 443, source: "import" },
            ]}
          />
        </MemoryRouter>
      </QueryClientProvider>
    </I18nextProvider>,
  )
}

describe("NodeSection batch toast", () => {
  it("publishes a densified batch summary with failed samples", async () => {
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path.includes("/api/nodes/test-batch")) {
        return Promise.resolve(new Response(JSON.stringify({
          results: [
            { tag: "hk-01", test_type: "tcp", success: true, latency_ms: 18 },
            { tag: "us-01", test_type: "tcp", success: false, error: "timeout" },
          ],
        })))
      }
      return Promise.resolve(new Response("{}"))
    }))
    const user = userEvent.setup()
    wrap()
    await user.click(screen.getByRole("button", { name: "批量测速" }))
    await waitFor(() => expect(toast.warning).toHaveBeenCalled())
    const message = String(vi.mocked(toast.warning).mock.calls[0][0])
    expect(message).toContain("1/2 成功")
    expect(message).toContain("最快 hk-01 18ms")
    expect(message).toContain("失败样例")
    expect(message).toContain("us-01/TCP: timeout")
  })
})
