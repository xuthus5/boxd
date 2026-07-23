import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"
import { MemoryRouter } from "react-router-dom"

import { RouteVisualEditor } from "@/features/policy/route-visual-editor"
import { i18n } from "@/i18n"

function renderEditor(route = "/policy/route") {
  return render(
    <I18nextProvider i18n={i18n}>
      <MemoryRouter initialEntries={[route]}>
        <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
          <RouteVisualEditor
            object={{
              rules: [
                { action: "route", outbound: "proxy", domain: ["google.com"] },
                { action: "route", outbound: "direct", domain: ["cn.example"] },
              ],
            }}
            revision={0}
            onChange={vi.fn()}
            onFieldValidityChange={vi.fn()}
            outbounds={[{ type: "direct", tag: "direct" }, { type: "selector", tag: "proxy" }]}
            metadata={[
              { name: "Google", description: "search" },
              { name: "CN", description: "local" },
            ]}
            onMetadataChange={vi.fn()}
          />
        </QueryClientProvider>
      </MemoryRouter>
    </I18nextProvider>,
  )
}

describe("route rule search", () => {
  it("filters visible rules by query", async () => {
    const user = userEvent.setup()
    renderEditor()
    expect(screen.getByText("Google")).toBeInTheDocument()
    expect(screen.getByText("CN")).toBeInTheDocument()
    await user.type(screen.getByLabelText("搜索规则"), "google")
    expect(screen.getByText("Google")).toBeInTheDocument()
    expect(screen.queryByText("CN")).not.toBeInTheDocument()
    expect(screen.getByText(/显示 1 \/ 2/)).toBeInTheDocument()
  })

  it("seeds rule search from deep-link query params", () => {
    renderEditor("/policy/route?q=google")
    expect(screen.getByLabelText("搜索规则")).toHaveValue("google")
    expect(screen.getByText("Google")).toBeInTheDocument()
    expect(screen.queryByText("CN")).not.toBeInTheDocument()
  })
})
