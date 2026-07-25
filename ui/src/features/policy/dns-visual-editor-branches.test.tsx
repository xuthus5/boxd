import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { useState } from "react"
import { I18nextProvider } from "react-i18next"
import { MemoryRouter } from "react-router-dom"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { toast } from "sonner"

import { DNSVisualEditor } from "@/features/policy/dns-visual-editor"
import type { JsonObject } from "@/features/policy/policy-form-model"
import { api } from "@/lib/api/endpoints"
import type { DNSProbeResult } from "@/lib/api/types"
import { i18n } from "@/i18n"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn(), message: vi.fn() } }))
vi.mock("@/features/policy/dns-global-card", () => ({
  DNSGlobalCard: () => <div data-testid="dns-global" />, DNSFakeIPCard: () => <div data-testid="dns-fakeip" />,
}))
vi.mock("@/features/policy/dns-server-card", () => ({
  DNSServerCard: ({ item, index, onEdit, onCopy, onDelete, onProbeResult }: {
    item: JsonObject; index: number; onEdit: () => void; onCopy: () => void; onDelete: () => void
    onProbeResult?: (result: DNSProbeResult) => void
  }) => <div data-testid={`dns-server-${index}`}><button onClick={onEdit}>edit server {index}</button>
    <button onClick={onCopy}>copy server {index}</button><button onClick={onDelete}>delete server {index}</button>
    <button onClick={() => onProbeResult?.({
      tag: typeof item.tag === "string" ? item.tag : undefined, type: String(item.type ?? "udp"), success: true, latency_ms: 1,
    })}>probe result {index}</button>
  </div>,
}))
vi.mock("@/features/policy/dns-rule-card", () => ({
  DNSRuleCard: ({ index, onEdit, onCopy, onMoveUp, onMoveDown, onDelete, onToggleInvert }: {
    index: number; onEdit: () => void; onCopy: () => void; onMoveUp: () => void; onMoveDown: () => void
    onDelete: () => void; onToggleInvert?: () => void
  }) => <div data-testid={`dns-rule-${index}`}><button onClick={onEdit}>edit dns rule {index}</button>
    <button onClick={onCopy}>copy dns rule {index}</button><button onClick={onMoveUp}>up dns rule {index}</button>
    <button onClick={onMoveDown}>down dns rule {index}</button><button onClick={onDelete}>delete dns rule {index}</button>
    {onToggleInvert ? <button onClick={onToggleInvert}>invert dns rule {index}</button> : null}
  </div>,
}))
vi.mock("@/features/policy/dns-visual-dialogs", () => ({
  DNSVisualDialogs: ({ selection, onClose, onClearJumpPath, onSave }: {
    selection: { kind: string; item: JsonObject } | null; onClose: () => void; onClearJumpPath: () => void
    onSave: (item: JsonObject) => void
  }) => selection ? <div data-testid="dns-dialog"><span>{selection.kind}</span>
    <button onClick={onClose}>close dns dialog</button><button onClick={onClearJumpPath}>clear dns jump</button>
    <button onClick={() => onSave({ ...selection.item, saved: true })}>save dns dialog</button>
  </div> : null,
}))

function renderDNS(ui: React.ReactElement, route = "/policy/dns") {
  return render(<I18nextProvider i18n={i18n}><MemoryRouter initialEntries={[route]}>
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })}>{ui}</QueryClientProvider>
  </MemoryRouter></I18nextProvider>)
}

function DNSHarness({ initial, onRulesChange, jumpPath, onJumpPathHandled }: {
  initial: JsonObject; onRulesChange?: (object: JsonObject, metadata: never[]) => void
  jumpPath?: string; onJumpPathHandled?: () => void
}) {
  const [object, setObject] = useState(initial)
  return <><DNSVisualEditor object={object} revision={0} onChange={setObject} onFieldValidityChange={vi.fn()}
    onRulesChange={onRulesChange} jumpPath={jumpPath} onJumpPathHandled={onJumpPathHandled} />
    <output aria-label="dns branch state">{JSON.stringify(object)}</output></>
}

describe("DNSVisualEditor uncovered branches", () => {
  beforeEach(() => vi.clearAllMocks())
  afterEach(() => vi.restoreAllMocks())

  it("renders empty sections and saves server and rule selections", async () => {
    renderDNS(<DNSHarness initial={{}} />)
    expect(screen.getByText("暂无 DNS 服务器")).toBeInTheDocument()
    expect(screen.getByText("暂无 DNS 规则")).toBeInTheDocument()
    await userEvent.click(screen.getAllByRole("button", { name: "新增 DNS 服务器" })[0])
    expect(screen.getByTestId("dns-dialog")).toHaveTextContent("server")
    await userEvent.click(screen.getByRole("button", { name: "save dns dialog" }))
    expect(screen.getByLabelText("dns branch state")).toHaveTextContent('"servers"')
    await userEvent.click(screen.getAllByRole("button", { name: "新增 DNS 规则" })[0])
    expect(screen.getByTestId("dns-dialog")).toHaveTextContent("rule")
    await userEvent.click(screen.getByRole("button", { name: "save dns dialog" }))
    expect(screen.getByLabelText("dns branch state")).toHaveTextContent('"rules"')
  })

  it("filters both lists and exercises server and rule callbacks", async () => {
    const onRulesChange = vi.fn()
    renderDNS(<DNSHarness onRulesChange={onRulesChange} initial={{
      servers: [{ tag: "cf", type: "udp", server: "1.1.1.1" }, { type: "local", tag: "local" }],
      rules: [{ action: "reject", domain: ["blocked.example"] }, { action: "route", server: "cf" }],
    }} />, "/policy/dns?sq=missing&rq=missing")
    expect(screen.getAllByText("无匹配项")).toHaveLength(2)
    for (const button of screen.getAllByRole("button", { name: "清空搜索" })) await userEvent.click(button)
    fireEvent.change(screen.getByLabelText("搜索 DNS 服务器"), { target: { value: "cf" } })
    fireEvent.change(screen.getByLabelText("搜索 DNS 规则"), { target: { value: "blocked" } })
    expect(screen.getAllByText("显示 1 / 2")).toHaveLength(2)
    await userEvent.click(screen.getByRole("button", { name: "probe result" }))
    await userEvent.click(screen.getByRole("button", { name: "edit server" }))
    await userEvent.click(screen.getByRole("button", { name: "save dns dialog" }))
    await userEvent.click(screen.getByRole("button", { name: "copy server" }))
    await userEvent.click(screen.getAllByRole("button", { name: "delete server" })[0])
    await userEvent.click(screen.getByRole("button", { name: "edit dns rule 0" }))
    await userEvent.click(screen.getByRole("button", { name: "save dns dialog" }))
    await userEvent.click(screen.getByRole("button", { name: "copy dns rule 0" }))
    await userEvent.click(screen.getByRole("button", { name: "up dns rule 0" }))
    await userEvent.click(screen.getByRole("button", { name: "down dns rule 0" }))
    await userEvent.click(screen.getByRole("button", { name: "invert dns rule 0" }))
    await userEvent.click(screen.getByRole("button", { name: "delete dns rule 0" }))
    expect(onRulesChange).toHaveBeenCalled()
  })

  it("probes only supported servers, handles batch errors, and jumps to a server", async () => {
    const success = { results: [{ tag: "cf", type: "udp", success: true, latency_ms: 9 }] }
    const probeMock = vi.spyOn(api.runtime, "probeDNSBatch").mockResolvedValue(success)
    const onJumpPathHandled = vi.fn()
    renderDNS(<DNSHarness initial={{ servers: [{ tag: "cf", type: "udp", server: "1.1.1.1" }, { type: "local" }] }}
      jumpPath="dns.servers[0].server" onJumpPathHandled={onJumpPathHandled} />)
    await waitFor(() => expect(screen.getByTestId("dns-dialog")).toHaveTextContent("server"))
    await userEvent.click(screen.getByRole("button", { name: "clear dns jump" }))
    expect(onJumpPathHandled).toHaveBeenCalled()
    await userEvent.click(screen.getByRole("button", { name: "close dns dialog" }))
    await userEvent.click(screen.getByRole("button", { name: "全部探测" }))
    await waitFor(() => expect(probeMock).toHaveBeenCalledWith([{ tag: "cf", type: "udp", server: "1.1.1.1" }], 6))
    expect(vi.mocked(toast.success)).toHaveBeenCalled()
    probeMock.mockRejectedValueOnce(new Error("network down"))
    await userEvent.click(screen.getByRole("button", { name: "全部探测" }))
    await waitFor(() => expect(vi.mocked(toast.error)).toHaveBeenCalled())
    expect(vi.mocked(toast.error).mock.calls.at(-1)?.[0]).toContain("network down")
  })
})
