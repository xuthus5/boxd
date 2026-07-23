import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"

import { RawConfigPage } from "@/features/advanced/raw-config-page"
import { i18n } from "@/i18n"

describe("RawConfigPage diff", () => {
  it("renders a no-change summary for the loaded config", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
      status: "ok",
      data: { log: { level: "info" } },
      error: null,
      meta: null,
    }))))
    render(
      <I18nextProvider i18n={i18n}>
        <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
          <RawConfigPage />
        </QueryClientProvider>
      </I18nextProvider>,
    )
    await waitFor(() => expect(screen.getByTestId("raw-config-diff")).toHaveTextContent("配置无变化"))
    expect(screen.getByLabelText("完整配置 JSON")).toBeInTheDocument()
    vi.unstubAllGlobals()
  })
})
