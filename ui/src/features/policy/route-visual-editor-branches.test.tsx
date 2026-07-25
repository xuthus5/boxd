import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { useState } from "react"
import { I18nextProvider } from "react-i18next"
import { MemoryRouter } from "react-router-dom"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { toast } from "sonner"

import { RouteVisualEditor } from "@/features/policy/route-visual-editor"
import type { JsonObject } from "@/features/policy/policy-form-model"
import { api } from "@/lib/api/endpoints"
import type { RouteRuleMetadata, RuleSetStatusItem, RuleSetUpdateResponse } from "@/lib/api/types"
import { i18n } from "@/i18n"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn(), message: vi.fn() } }))
vi.mock("@/features/policy/route-global-card", () => ({
  RouteGlobalCard: ({ onInstall }: { onInstall?: () => void }) => <button data-testid="route-install" onClick={onInstall}>install</button>,
}))
vi.mock("@/features/policy/route-rule-card", () => ({
  RouteRuleCard: ({ index, onEdit, onCopy, onMoveUp, onMoveDown, onDelete, onToggleInvert }: {
    index: number; onEdit: () => void; onCopy: () => void; onMoveUp: () => void; onMoveDown: () => void
    onDelete: () => void; onToggleInvert?: () => void
  }) => <div data-testid={`route-rule-${index}`}>
    <button onClick={onEdit}>edit rule {index}</button><button onClick={onCopy}>copy rule {index}</button>
    <button onClick={onMoveUp}>up rule {index}</button><button onClick={onMoveDown}>down rule {index}</button>
    <button onClick={onDelete}>delete rule {index}</button>
    {onToggleInvert ? <button onClick={onToggleInvert}>invert rule {index}</button> : null}
  </div>,
}))
vi.mock("@/features/policy/route-rule-set-card", () => ({
  RouteRuleSetCard: ({ item, lastUpdate, updating, onEdit, onCopy, onDelete, onUpdate }: {
    item: JsonObject; lastUpdate?: { error?: string }; updating?: boolean; onEdit: () => void
    onCopy: () => void; onDelete: () => void; onUpdate?: () => void
  }) => <div data-testid={`route-set-${String(item.tag ?? "unnamed")}`}>
    <span>{lastUpdate?.error ?? "clean"}</span><button onClick={onEdit}>edit set</button>
    <button onClick={onCopy}>copy set</button><button onClick={onDelete}>delete set</button>
    {onUpdate ? <button disabled={updating} onClick={onUpdate}>update set</button> : null}
  </div>,
}))
vi.mock("@/features/policy/route-visual-dialogs", () => ({
  RouteVisualDialogs: ({ selection, onClose, onClearJumpPath, onSave }: {
    selection: { kind: string; item: JsonObject } | null; onClose: () => void; onClearJumpPath: () => void
    onSave: (item: JsonObject, metadata?: RouteRuleMetadata) => void
  }) => selection ? <div data-testid="route-dialog"><span>{selection.kind}</span>
    <button onClick={onClose}>close dialog</button><button onClick={onClearJumpPath}>clear jump</button>
    <button onClick={() => onSave({ ...selection.item, saved: true }, selection.kind === "rule" ? { name: "saved", description: "" } : undefined)}>save dialog</button>
  </div> : null,
}))
vi.mock("@/features/common/page-load-error-alert", () => ({
  PageLoadErrorAlert: ({ onRetry }: { onRetry?: () => void }) => <button data-testid="route-metadata-retry" onClick={onRetry}>retry metadata</button>,
}))

function renderRoute(ui: React.ReactElement, route = "/policy/route") {
  return render(<I18nextProvider i18n={i18n}><MemoryRouter initialEntries={[route]}>
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })}>{ui}</QueryClientProvider>
  </MemoryRouter></I18nextProvider>)
}

function RouteHarness({ initial, metadata = [], metadataLoading, metadataError, onMetadataRetry, onRulesChange, jumpPath, onJumpPathHandled }: {
  initial: JsonObject; metadata?: RouteRuleMetadata[]; metadataLoading?: boolean; metadataError?: unknown
  onMetadataRetry?: () => void; onRulesChange?: (object: JsonObject, metadata: RouteRuleMetadata[]) => void
  jumpPath?: string; onJumpPathHandled?: () => void
}) {
  const [object, setObject] = useState(initial)
  const [currentMetadata, setCurrentMetadata] = useState(metadata)
  return <><RouteVisualEditor object={object} revision={0} onChange={setObject} onFieldValidityChange={vi.fn()}
    metadata={currentMetadata} metadataLoading={metadataLoading} metadataError={metadataError} onMetadataRetry={onMetadataRetry}
    onMetadataChange={setCurrentMetadata} onRulesChange={onRulesChange} jumpPath={jumpPath} onJumpPathHandled={onJumpPathHandled} />
    <output aria-label="route branch state">{JSON.stringify(object)}</output>
    <output aria-label="route branch metadata">{JSON.stringify(currentMetadata)}</output></>
}

const updateEnvelope = (data: RuleSetUpdateResponse) => ({ status: "ok" as const, data, error: null, meta: null })

describe("RouteVisualEditor uncovered branches", () => {
  beforeEach(() => vi.clearAllMocks())
  afterEach(() => vi.restoreAllMocks())

  it("renders loading, error, empty sections, and selection saves", async () => {
    const loading = renderRoute(<RouteHarness initial={{}} metadataLoading />)
    expect(document.querySelector('[data-slot="skeleton"]')).toBeInTheDocument()
    loading.unmount()

    const retry = vi.fn()
    const errored = renderRoute(<RouteHarness initial={{}} metadataError={new Error("metadata")} onMetadataRetry={retry} />)
    await userEvent.click(screen.getByTestId("route-metadata-retry"))
    expect(retry).toHaveBeenCalledOnce()
    errored.unmount()

    renderRoute(<RouteHarness initial={{}} />)
    expect(screen.getByText("暂无路由规则")).toBeInTheDocument()
    expect(screen.getByText("暂无路由规则集")).toBeInTheDocument()
    await userEvent.click(screen.getAllByRole("button", { name: "新增规则" })[0])
    expect(screen.getByTestId("route-dialog")).toHaveTextContent("rule")
    await userEvent.click(screen.getByRole("button", { name: "save dialog" }))
    expect(screen.getByLabelText("route branch state")).toHaveTextContent('"saved":true')
  })

  it("filters rules and exercises every rule list callback", async () => {
    const onRulesChange = vi.fn()
    renderRoute(<RouteHarness onRulesChange={onRulesChange} initial={{
      rules: [{ action: "reject", domain: ["blocked.example"] }, { action: "route", outbound: "proxy" }],
      rule_set: [{ type: "inline", tag: "geo" }],
    }} />, "/policy/route?q=missing")
    expect(screen.getByText("无匹配规则")).toBeInTheDocument()
    await userEvent.click(screen.getByRole("button", { name: "清空搜索" }))
    expect(screen.getByTestId("route-rule-0")).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText("搜索规则"), { target: { value: "blocked" } })
    expect(screen.getByText("显示 1 / 2")).toBeInTheDocument()
    await userEvent.click(screen.getByRole("button", { name: /reject/ }))
    await userEvent.click(screen.getByRole("button", { name: "edit rule 0" }))
    await userEvent.click(screen.getByRole("button", { name: "save dialog" }))
    await userEvent.click(screen.getByRole("button", { name: "copy rule 0" }))
    await userEvent.click(screen.getByRole("button", { name: "up rule 0" }))
    await userEvent.click(screen.getByRole("button", { name: "down rule 0" }))
    await userEvent.click(screen.getByRole("button", { name: "invert rule 0" }))
    await userEvent.click(screen.getByRole("button", { name: "delete rule 0" }))
    expect(onRulesChange).toHaveBeenCalled()
  })

  it("updates rule sets, handles success and request errors, and jumps by path", async () => {
    const status: RuleSetStatusItem[] = [{ tag: "geo", type: "remote", builtin: false, updatable: true, note: "ready" }]
    const response: RuleSetUpdateResponse = {
      results: [{ tag: "geo", type: "remote", ok: false, error: "connection refused", error_code: "network" }],
      updated_count: 0, failed_count: 1, skipped_count: 0, restarted: false,
    }
    const statusMock = vi.spyOn(api.config, "ruleSetsStatus").mockResolvedValue(status)
    const updateMock = vi.spyOn(api.config, "updateRuleSets").mockResolvedValue(updateEnvelope(response))
    const onJumpPathHandled = vi.fn()
    renderRoute(<RouteHarness initial={{ rules: [{ action: "reject" }], rule_set: [{ type: "remote", tag: "geo" }] }}
      jumpPath="route.rules[0].action" onJumpPathHandled={onJumpPathHandled} />)
    await waitFor(() => expect(statusMock).toHaveBeenCalled())
    expect(screen.getByTestId("route-dialog")).toHaveTextContent("rule")
    await userEvent.click(screen.getByRole("button", { name: "clear jump" }))
    expect(onJumpPathHandled).toHaveBeenCalled()
    await userEvent.click(screen.getByRole("button", { name: "close dialog" }))
    await userEvent.click(screen.getByRole("button", { name: "update set" }))
    await waitFor(() => expect(updateMock).toHaveBeenCalledWith({ tags: ["geo"] }))
    expect(screen.getByTestId("route-set-geo")).toHaveTextContent("connection refused")
    updateMock.mockRejectedValueOnce(new Error("network down"))
    await userEvent.click(screen.getByRole("button", { name: "update set" }))
    await waitFor(() => expect(vi.mocked(toast.error)).toHaveBeenCalled())
    expect(vi.mocked(toast.error).mock.calls.at(-1)?.[0]).toContain("network down")
  })
})
