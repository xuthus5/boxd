import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"
import { I18nextProvider } from "react-i18next"

import { NodeCard } from "@/features/nodes/node-card"
import { i18n } from "@/i18n"

describe("NodeCard latency tone", () => {
  it("renders colored latency badges for success results", () => {
    render(
      <I18nextProvider i18n={i18n}>
        <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
          <NodeCard
            node={{ tag: "hk", type: "vless", server: "hk.example", port: 443, source: "import" }}
            results={{
              tcp: { tag: "hk", test_type: "tcp", success: true, latency_ms: 40 },
              http: { tag: "hk", test_type: "http", success: true, latency_ms: 220 },
              icmp: { tag: "hk", test_type: "icmp", success: false, error: "timeout" },
            }}
          />
        </QueryClientProvider>
      </I18nextProvider>,
    )
    expect(screen.getByText("40 ms")).toBeInTheDocument()
    expect(screen.getByText("220 ms")).toBeInTheDocument()
    expect(screen.getByText("timeout")).toBeInTheDocument()
  })
})
