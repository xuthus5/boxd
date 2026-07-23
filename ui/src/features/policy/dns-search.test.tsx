import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"
import { MemoryRouter } from "react-router-dom"

import { DNSVisualEditor } from "@/features/policy/dns-visual-editor"
import { i18n } from "@/i18n"

function renderEditor(route = "/policy/dns") {
  return render(
    <I18nextProvider i18n={i18n}>
      <MemoryRouter initialEntries={[route]}>
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
      </MemoryRouter>
    </I18nextProvider>,
  )
}

describe("DNS search", () => {
  it("filters servers and rules by query", async () => {
    const user = userEvent.setup()
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ outbounds: [] }))))
    renderEditor()
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

  it("seeds server and rule filters from deep-link query params", () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ outbounds: [] }))))
    renderEditor("/policy/dns?sq=remote&rq=ads")
    expect(screen.getByLabelText("搜索 DNS 服务器")).toHaveValue("remote")
    expect(screen.getByLabelText("搜索 DNS 规则")).toHaveValue("ads")
    expect(screen.getByText("dns-remote")).toBeInTheDocument()
    expect(screen.queryByText("dns-local")).not.toBeInTheDocument()
    expect(screen.getByText("ads.example")).toBeInTheDocument()
    expect(screen.queryByText("google.com")).not.toBeInTheDocument()
    vi.unstubAllGlobals()
  })
  it("clears server search from empty-state action", async () => {
    const user = userEvent.setup()
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ outbounds: [] }))))
    renderEditor("/policy/dns?sq=missing-server")
    expect(await screen.findByText("无匹配项")).toBeInTheDocument()
    await user.click(screen.getAllByRole("button", { name: "清空搜索" })[0])
    expect(await screen.findByText("dns-remote")).toBeInTheDocument()
    expect(screen.getByText("dns-local")).toBeInTheDocument()
    expect(screen.getByLabelText("搜索 DNS 服务器")).toHaveValue("")
    vi.unstubAllGlobals()
  })

})
