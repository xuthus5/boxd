import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type { ReactElement } from "react"
import { afterEach, describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"
import { MemoryRouter } from "react-router-dom"
import { toast } from "sonner"

import { DNSServerCard } from "@/features/policy/dns-server-card"
import { i18n } from "@/i18n"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

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
    vi.mocked(toast.success).mockClear()
    vi.mocked(toast.error).mockClear()
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: undefined })
  })

  function stubClipboard(writeText = vi.fn().mockResolvedValue(undefined)) {
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } })
    return writeText
  }

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

  it("renders normal and coded failures and copies probe errors by keyboard", async () => {
    const writeText = stubClipboard()
    renderCard(
      <DNSServerCard
        item={{ tag: "cf", type: "udp", server: "1.1.1.1" }}
        onEdit={vi.fn()}
        onCopy={vi.fn()}
        onDelete={vi.fn()}
        probeResult={{ tag: "cf", type: "udp", success: false, error: "connection refused", error_code: "network" }}
      />,
    )
    const expectedLabel = "复制探测错误: cf connection refused"
    const badge = screen.getByRole("button", { name: expectedLabel })
    expect(badge).toHaveTextContent("network: connection refused")
    fireEvent.keyDown(badge, { key: "Escape" })
    fireEvent.keyDown(badge, { key: "Enter" })
    fireEvent.keyDown(badge, { key: " " })
    await waitFor(() => expect(writeText).toHaveBeenCalled())
    expect(vi.mocked(toast.success)).toHaveBeenCalledWith("探测错误已复制")

    renderCard(
      <DNSServerCard
        item={{ tag: "normal", type: "udp", server: "1.1.1.1" }}
        onEdit={vi.fn()}
        onCopy={vi.fn()}
        onDelete={vi.fn()}
        probeResult={{ tag: "normal", type: "udp", success: true }}
      />,
    )
    expect(screen.getByText("正常", { selector: '[data-slot="badge"]' })).toBeInTheDocument()
  })

  it("surfaces failed probe responses and request errors", async () => {
    stubClipboard()
    const onProbeResult = vi.fn()
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
      status: "ok",
      data: { tag: "cf", type: "udp", success: false, error: "no response", error_code: "no_response" },
      error: null,
    }), { status: 200, headers: { "Content-Type": "application/json" } })))
    renderCard(
      <DNSServerCard item={{ tag: "cf", type: "udp", server: "1.1.1.1" }} onEdit={vi.fn()} onCopy={vi.fn()} onDelete={vi.fn()} onProbeResult={onProbeResult} />,
    )
    await userEvent.click(screen.getByRole("button", { name: /探测 DNS 服务器 cf/ }))
    await waitFor(() => expect(onProbeResult).toHaveBeenCalledWith(expect.objectContaining({ success: false })))
    expect(vi.mocked(toast.error)).toHaveBeenCalled()

    vi.mocked(toast.error).mockClear()
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("network down")))
    await userEvent.click(screen.getByRole("button", { name: /探测 DNS 服务器 cf/ }))
    await waitFor(() => expect(vi.mocked(toast.error)).toHaveBeenCalled())
    const lastCall = vi.mocked(toast.error).mock.calls.at(-1)
    const options = lastCall?.[1] as { action?: { onClick: () => void } } | undefined
    expect(String(lastCall?.[0])).toContain("network down")
    options?.action?.onClick()
    await waitFor(() => expect(vi.mocked(toast.success)).toHaveBeenCalledWith("探测错误已复制"))
  })

  it("handles empty probe payloads and mobile delete cancellation", async () => {
    const writeText = stubClipboard()
    const onDelete = vi.fn()
    renderCard(
      <DNSServerCard
        item={{ tag: "cf", type: "udp", server: "1.1.1.1" }}
        onEdit={vi.fn()}
        onCopy={vi.fn()}
        onDelete={onDelete}
        probeResult={{ success: false }}
      />,
    )
    await userEvent.click(screen.getByRole("button", { name: /复制探测错误:/ }))
    expect(writeText).not.toHaveBeenCalled()

    await userEvent.click(screen.getByRole("button", { name: /更多 DNS 服务器/ }))
    await userEvent.click(await screen.findByRole("menuitem", { name: "删除" }))
    await userEvent.click(screen.getByRole("button", { name: "取消" }))
    expect(onDelete).not.toHaveBeenCalled()
  })
})
