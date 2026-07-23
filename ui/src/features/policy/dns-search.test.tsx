import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"

import { DNSVisualEditor } from "@/features/policy/dns-visual-editor"
import { i18n } from "@/i18n"

describe("DNS search", () => {
  it("filters servers and rules by query", async () => {
    const user = userEvent.setup()
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ outbounds: [] }))))
    render(
      <I18nextProvider i18n={i18n}>
        <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
          <DNSVisualEditor
            object={{
              servers: [
                { tag: "dns-remote", type: "https", server: "1.1.1.1" },
                { tag: "dns-local", type: "local" },
              ],
              rules: [
                { action: "route", server: "dns-remote", domain: ["google.com"] },
                { action: "reject", domain: ["ads.example"] },
              ],
            }}
            revision={0}
            onChange={vi.fn()}
            onFieldValidityChange={vi.fn()}
          />
        </QueryClientProvider>
      </I18nextProvider>,
    )
    expect(screen.getByText("dns-remote")).toBeInTheDocument()
    expect(screen.getByText("dns-local")).toBeInTheDocument()
    await user.type(screen.getByLabelText("搜索 DNS 服务器"), "remote")
    expect(screen.getByText("dns-remote")).toBeInTheDocument()
    expect(screen.queryByText("dns-local")).not.toBeInTheDocument()

    await user.type(screen.getByLabelText("搜索 DNS 规则"), "ads")
    expect(screen.getByText("ads.example")).toBeInTheDocument()
    expect(screen.queryByText("google.com")).not.toBeInTheDocument()
    vi.unstubAllGlobals()
  })
})
