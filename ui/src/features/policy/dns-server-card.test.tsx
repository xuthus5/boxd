import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type { ReactElement } from "react"
import { afterEach, describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"
import { MemoryRouter } from "react-router-dom"

import { DNSServerCard } from "@/features/policy/dns-server-card"
import { i18n } from "@/i18n"

function renderCard(ui: ReactElement) {
  return render(
    <I18nextProvider i18n={i18n}>
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })}>
        <MemoryRouter>{ui}</MemoryRouter>
      </QueryClientProvider>
    </I18nextProvider>,
  )
}

describe("DNSServerCard", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it("renders unnamed server and confirms delete", async () => {
    const onEdit = vi.fn()
    const onCopy = vi.fn()
    const onDelete = vi.fn()
    renderCard(
      <DNSServerCard item={{ type: "udp", server: "1.1.1.1" }} onEdit={onEdit} onCopy={onCopy} onDelete={onDelete} />,
    )
    expect(screen.getByText("未命名")).toBeInTheDocument()
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: /编辑 DNS 服务器/ }))
    expect(onEdit).toHaveBeenCalled()
    await user.click(screen.getByRole("button", { name: /复制 DNS 服务器/ }))
    expect(onCopy).toHaveBeenCalled()
    await user.click(screen.getByRole("button", { name: /删除 DNS 服务器/ }))
    await user.click(screen.getByRole("button", { name: "确认删除" }))
    expect(onDelete).toHaveBeenCalled()
  })

  it("probes server and surfaces latency", async () => {
    const onProbeResult = vi.fn()
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
      status: "ok",
      data: { tag: "cf", type: "udp", success: true, latency_ms: 12 },
      error: null,
    }), { status: 200, headers: { "Content-Type": "application/json" } })))
    renderCard(
      <DNSServerCard
        item={{ tag: "cf", type: "udp", server: "1.1.1.1" }}
        onEdit={vi.fn()}
        onCopy={vi.fn()}
        onDelete={vi.fn()}
        onProbeResult={onProbeResult}
      />,
    )
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: /探测 DNS 服务器 cf/ }))
    await waitFor(() => expect(onProbeResult).toHaveBeenCalledWith(expect.objectContaining({
      tag: "cf",
      success: true,
      latency_ms: 12,
    })))
  })

  it("disables probe for local servers", () => {
    renderCard(
      <DNSServerCard item={{ tag: "sys", type: "local" }} onEdit={vi.fn()} onCopy={vi.fn()} onDelete={vi.fn()} />,
    )
    expect(screen.getByRole("button", { name: /探测 DNS 服务器 sys/ })).toBeDisabled()
  })

  it("deep-links tagged servers to DNS rules and logs", () => {
    renderCard(
      <DNSServerCard item={{ tag: "cf", type: "udp", server: "1.1.1.1" }} onEdit={vi.fn()} onCopy={vi.fn()} onDelete={vi.fn()} />,
    )
    expect(screen.getByRole("link", { name: "查看 DNS 规则: cf" })).toHaveAttribute("href", "/policy/dns?rq=cf")
    expect(screen.getByRole("link", { name: "查看日志: cf" })).toHaveAttribute(
      "href",
      "/observability/logs?q=cf&preset=dns",
    )
  })
})
