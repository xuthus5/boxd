import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"
import { toast } from "sonner"

import { SetupDialog } from "@/features/setup/setup-dialog"
import { setupStatus } from "@/test/setup-fixtures"
import { renderApp } from "@/test/render"
import type { SetupStatus } from "@/lib/api/setup"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn(), message: vi.fn() } }))
afterEach(() => { vi.unstubAllGlobals(); vi.clearAllMocks() })

const preview = { source_hash: "preview-hash", current_config: {}, config: { inbounds: [{ type: "mixed", tag: "mixed-in", listen: "0.0.0.0", listen_port: 1080 }] },
  modules: ["inbounds", "dns", "outbounds"], warnings: ["Check host port publication"], will_restart: false }

function renderDialog(status: SetupStatus = setupStatus()) {
  const onClose = vi.fn()
  renderApp(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
    <SetupDialog status={status} onClose={onClose} />
  </QueryClientProvider>)
  return onClose
}

function mockRequests(outcome = "ok") {
  const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
    if (url.endsWith("/preview")) return new Response(JSON.stringify(preview))
    if (url.endsWith("/apply")) return new Response(JSON.stringify({ status: outcome,
      data: { modules: preview.modules, config_hash: "new" }, error: outcome === "rolled_back" ? { code: "config_restart_failed", message: "previous configuration restored" } : null,
      meta: { rolled_back: outcome === "rolled_back" } }))
    throw new Error(`unexpected request ${url} ${init?.method}`)
  })
  vi.stubGlobal("fetch", fetchMock)
  return fetchMock
}

describe("SetupDialog", () => {
  it("requires a preview, uses its revision and never starts a stopped kernel", async () => {
    const fetchMock = mockRequests()
    const close = renderDialog()
    const user = userEvent.setup()
    expect(screen.getByRole("button", { name: "应用此配置" })).toBeDisabled()
    expect(screen.getByRole("button", { name: "TUN 接管" })).toBeDisabled()
    expect(screen.getByText("/dev/net/tun unavailable")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "预览变更" }))
    expect(await screen.findByTestId("config-diff-panel")).toHaveTextContent("inbounds")
    expect(screen.getByText("Check host port publication")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "应用此配置" }))
    await waitFor(() => expect(close).toHaveBeenCalledOnce())
    expect(fetchMock).toHaveBeenCalledWith("/api/config/setup/apply", expect.objectContaining({ body: expect.stringContaining('"source_hash":"preview-hash"') }))
    expect(fetchMock.mock.calls.some(([url]) => url.includes("/service/start"))).toBe(false)
  })

  it("invalidates the preview when module or IPv6 choices change", async () => {
    mockRequests()
    renderDialog()
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "预览变更" }))
    await screen.findByTestId("config-diff-panel")
    await user.click(screen.getByRole("button", { name: "关闭 IPv6" }))
    expect(screen.queryByTestId("config-diff-panel")).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: "应用此配置" })).toBeDisabled()
  })

  it("keeps the dialog open and reports rolled back apply without success", async () => {
    mockRequests("rolled_back")
    const close = renderDialog()
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "预览变更" }))
    await screen.findByTestId("config-diff-panel")
    await user.click(screen.getByRole("button", { name: "应用此配置" }))
    expect(await screen.findByTestId("config-save-error")).toHaveTextContent("配置已回滚")
    expect(toast.success).not.toHaveBeenCalled()
    expect(close).not.toHaveBeenCalled()
    expect(screen.getByRole("button", { name: "应用此配置" })).toBeDisabled()
  })

  it("requires explicit consent before rebuilding invalid JSON", async () => {
    const fetchMock = mockRequests()
    renderDialog(setupStatus({ config_error: "invalid JSON" }))
    const user = userEvent.setup()
    expect(screen.getByRole("button", { name: "预览变更" })).toBeDisabled()
    await user.click(screen.getByRole("checkbox", { name: "我确认用所选默认模块重建无效配置" }))
    await user.click(screen.getByRole("button", { name: "预览变更" }))
    await screen.findByTestId("config-diff-panel")
    expect(fetchMock).toHaveBeenCalledWith("/api/config/setup/preview", expect.objectContaining({ body: expect.stringContaining('"reset_invalid":true') }))
  })

  it.each(["on", "off"])("allows explicit IPv6 %s when capability is unconfirmed", async (mode) => {
    const fetchMock = mockRequests()
    const status = setupStatus()
    status.capabilities.ipv6_reason = "ipv6_unknown"
    renderDialog(status)
    const user = userEvent.setup()
    expect(screen.getByRole("button", { name: "启用 IPv6" })).toBeEnabled()
    await user.click(screen.getByRole("button", { name: mode === "on" ? "启用 IPv6" : "关闭 IPv6" }))
    await user.click(screen.getByRole("button", { name: "预览变更" }))
    await screen.findByTestId("config-diff-panel")
    expect(JSON.parse(fetchMock.mock.calls[0]![1]!.body as string).ipv6).toBe(mode)
  })

  it.each(["ipv6_disabled", "ipv6_unsupported"])("keeps explicit IPv6 on disabled for %s", async (reason) => {
    mockRequests()
    const status = setupStatus()
    status.capabilities.ipv6_reason = reason
    renderDialog(status)
    expect(screen.getByRole("button", { name: "启用 IPv6" })).toBeDisabled()
    expect(screen.getByRole("button", { name: "关闭 IPv6" })).toBeEnabled()
  })

  it("allows an explicitly confirmed repair of semantic configuration errors", async () => {
    const fetchMock = mockRequests()
    renderDialog(setupStatus({ config_error: "dns.servers[0].detour: unknown outbound proxy" }))
    const user = userEvent.setup()
    expect(screen.getByText("dns.servers[0].detour: unknown outbound proxy")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "预览变更" })).toBeDisabled()
    await user.click(screen.getByRole("checkbox", { name: "我确认用所选默认模块重建无效配置" }))
    await user.click(screen.getByRole("button", { name: "预览变更" }))
    await screen.findByTestId("config-diff-panel")
    expect(JSON.parse(fetchMock.mock.calls[0]![1]!.body as string).reset_invalid).toBe(true)
  })

  it("shows available capabilities and previews explicit TUN with IPv6", async () => {
    const fetchMock = mockRequests()
    const status = setupStatus()
    status.capabilities = { ...status.capabilities, container: false, tun_available: true, ipv6_available: true }
    status.listeners = [{ tag: "mixed", type: "mixed", listen: "127.0.0.1", listen_port: 1080 }]
    status.modules[0] = { ...status.modules[0]!, state: "invalid", dependencies: ["route"], message: "listener needs repair" }
    renderDialog(status)
    const user = userEvent.setup()
    expect(screen.getByText("TUN 可用")).toBeInTheDocument()
    expect(screen.getByText("IPv6 可用")).toBeInTheDocument()
    expect(screen.getByText("listener needs repair")).toBeInTheDocument()
    expect(screen.getByText("依赖：路由")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "TUN 接管" }))
    await user.click(screen.getByRole("button", { name: "启用 IPv6" }))
    await user.click(screen.getByRole("button", { name: "预览变更" }))
    await screen.findByTestId("config-diff-panel")
    const body = JSON.parse(fetchMock.mock.calls[0]![1]!.body as string)
    expect(body).toMatchObject({ inbound_mode: "tun", ipv6: "on" })
  })

  it("prevents an empty selection from accidentally requesting all defaults", async () => {
    const fetchMock = mockRequests()
    renderDialog()
    const user = userEvent.setup()
    for (const checkbox of screen.getAllByRole("checkbox")) await user.click(checkbox)
    expect(screen.getByRole("button", { name: "预览变更" })).toBeDisabled()
    await user.click(screen.getByRole("checkbox", { name: "DNS" }))
    await user.click(screen.getByRole("button", { name: "预览变更" }))
    await screen.findByTestId("config-diff-panel")
    expect(JSON.parse(fetchMock.mock.calls[0]![1]!.body as string).modules).toEqual(["dns"])
  })

  it("reports a failed preview and can retry without applying", async () => {
    const fetchMock = mockRequests()
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ status: "error", data: null,
      error: { code: "config_invalid_runtime", message: "DNS dependency missing" }, meta: null }), { status: 400 }))
    renderDialog()
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "预览变更" }))
    expect(await screen.findByTestId("config-save-error")).toHaveTextContent("DNS dependency missing")
    expect(screen.getByRole("button", { name: "应用此配置" })).toBeDisabled()
    await user.click(screen.getByRole("button", { name: "关闭提示" }))
    await user.click(screen.getByRole("button", { name: "预览变更" }))
    expect(await screen.findByTestId("config-diff-panel")).toBeInTheDocument()
  })

  it("requires a new preview after a stale configuration conflict", async () => {
    const fetchMock = mockRequests()
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify(preview)))
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ status: "error", data: null,
      error: { code: "config_setup_conflict", message: "configuration changed; preview again" }, meta: null }), { status: 409 }))
    const close = renderDialog()
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "预览变更" }))
    await screen.findByTestId("config-diff-panel")
    await user.click(screen.getByRole("button", { name: "应用此配置" }))
    expect(await screen.findByTestId("config-save-error")).toHaveTextContent("configuration changed; preview again")
    expect(screen.queryByTestId("config-diff-panel")).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: "应用此配置" })).toBeDisabled()
    expect(close).not.toHaveBeenCalled()
  })

  it("explains restarts for a running kernel and supports keeping inbounds", async () => {
    const fetchMock = mockRequests()
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ ...preview, will_restart: true, warnings: [] })))
    const close = renderDialog()
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "保留现有入站" }))
    await user.click(screen.getByRole("button", { name: "预览变更" }))
    expect(await screen.findByText(/应用配置将重启内核/)).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "应用此配置" }))
    await waitFor(() => expect(close).toHaveBeenCalledOnce())
    expect(toast.success).toHaveBeenCalledWith("配置已应用")
  })

  it("supports cancel and Escape without applying configuration", async () => {
    const fetchMock = mockRequests()
    const close = renderDialog()
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "取消" }))
    await user.keyboard("{Escape}")
    expect(close).toHaveBeenCalledTimes(2)
    expect(fetchMock).not.toHaveBeenCalled()
  })
})
