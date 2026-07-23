import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"

import { RawConfigPage } from "@/features/advanced/raw-config-page"
import { i18n } from "@/i18n"

function renderPage() {
  return render(
    <I18nextProvider i18n={i18n}>
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })}>
        <RawConfigPage />
      </QueryClientProvider>
    </I18nextProvider>,
  )
}

describe("RawConfigPage diff", () => {
  it("renders a no-change panel for the loaded config", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
      status: "ok",
      data: { log: { level: "info" } },
      error: null,
      meta: null,
    }))))
    renderPage()
    await waitFor(() => expect(screen.getByTestId("config-diff-panel")).toHaveTextContent("配置无变化"))
    expect(screen.getByLabelText("完整配置 JSON")).toBeInTheDocument()
    vi.unstubAllGlobals()
  })

  it("shows path-level diff details after edits", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
      status: "ok",
      data: { log: { level: "info" } },
      error: null,
      meta: null,
    }))))
    renderPage()
    const editor = await screen.findByLabelText("完整配置 JSON")
    await userEvent.click(editor)
    await userEvent.keyboard("{Control>}a{/Control}")
    await userEvent.paste(JSON.stringify({ log: { level: "debug" }, dns: { final: "local" } }, null, 2))
    await waitFor(() => {
      expect(screen.getByTestId("config-diff-panel")).toHaveTextContent("log.level")
      expect(screen.getByTestId("config-diff-panel")).toHaveTextContent("dns")
    })
    vi.unstubAllGlobals()
  })
})
