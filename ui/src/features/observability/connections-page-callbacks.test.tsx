import type { ReactNode } from "react"

import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { I18nextProvider } from "react-i18next"
import { MemoryRouter } from "react-router-dom"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { ConnectionsPage } from "@/features/observability/connections-page"
import { i18n } from "@/i18n"

const state = vi.hoisted(() => {
  const connections = [
    {
      id: 1,
      target: "example.com:443",
      outbound: "proxy",
      rule: "geosite-google",
      network: "tcp",
      inbound: "mixed-in",
      protocol: "tls",
      process: "/usr/bin/curl",
      upload: 10,
      download: 20,
      start: "2026-07-24T00:00:00Z",
    },
    {
      id: 2,
      target: "cdn.example.net:443",
      outbound: "direct",
      rule: "geoip-cn",
      network: "udp",
      upload: 1,
      download: 2,
      start: "2026-07-24T00:01:00Z",
    },
  ]
  return {
    connections,
    stream: {
      items: [{ list: connections }],
      error: null as Error | null,
      status: "open",
      paused: false,
      reconnect: vi.fn(),
      setPaused: vi.fn(),
    },
    close: {
      connections,
      closingId: null,
      bulkBusy: false,
      closeOne: vi.fn(),
      closeAll: vi.fn(),
      closeGroup: vi.fn(),
      closeFiltered: vi.fn(),
    },
    downloadTextFile: vi.fn(),
    toastSuccess: vi.fn(),
    toastError: vi.fn(),
  }
})

vi.mock("sonner", () => ({
  toast: { success: state.toastSuccess, error: state.toastError },
}))

vi.mock("@/features/auth/auth-context", () => ({
  useAuth: () => ({ session: { token: "token" } }),
}))

vi.mock("@/features/observability/use-stream-buffer", () => ({
  useStreamBuffer: () => state.stream,
}))

vi.mock("@/features/observability/use-connection-close-actions", () => ({
  useConnectionCloseActions: () => state.close,
}))

vi.mock("@/features/observability/log-export", () => ({
  downloadTextFile: state.downloadTextFile,
}))

vi.mock("@/components/confirm-action", () => ({
  ConfirmAction: ({ trigger, onConfirm }: { trigger: ReactNode; onConfirm: () => void | Promise<void> }) => (
    <div>
      {trigger}
      <button type="button" data-testid="confirm-action" onClick={() => { void onConfirm() }}>confirm</button>
    </div>
  ),
}))

vi.mock("@/features/observability/connection-toolbar", () => ({
  ConnectionToolbar: (props: {
    canExport: boolean
    onQueryChange: (value: string) => void
    onNetworkChange: (value: string) => void
    onProtocolChange: (value: string) => void
    onInboundChange: (value: string) => void
    onOutboundChange: (value: string) => void
    onRuleChange: (value: string) => void
    onProcessChange: (value: string) => void
    onSortChange: (value: "target" | "traffic") => void
    onToggleColumn: (id: string, enabled: boolean) => void
    onClearFacets: () => void
    onTogglePause: () => void
    onExport: () => void
    onCloseFiltered: () => void
  }) => (
    <div>
      <button type="button" data-testid="query" onClick={() => props.onQueryChange("needle")}>query</button>
      <button type="button" data-testid="empty-filters" onClick={() => {
        props.onQueryChange("")
        props.onNetworkChange("")
        props.onProtocolChange("")
        props.onInboundChange("")
        props.onOutboundChange("")
        props.onRuleChange("")
        props.onProcessChange("")
      }}>empty filters</button>
      <button type="button" data-testid="network" onClick={() => props.onNetworkChange("tcp")}>network</button>
      <button type="button" data-testid="protocol" onClick={() => props.onProtocolChange("tls")}>protocol</button>
      <button type="button" data-testid="inbound" onClick={() => props.onInboundChange("mixed-in")}>inbound</button>
      <button type="button" data-testid="outbound" onClick={() => props.onOutboundChange("proxy")}>outbound</button>
      <button type="button" data-testid="rule" onClick={() => props.onRuleChange("geosite-google")}>rule</button>
      <button type="button" data-testid="process" onClick={() => props.onProcessChange("/usr/bin/curl")}>process</button>
      <button type="button" data-testid="sort-target" onClick={() => props.onSortChange("target")}>sort target</button>
      <button type="button" data-testid="sort-traffic" onClick={() => props.onSortChange("traffic")}>sort traffic</button>
      <button type="button" data-testid="column-on" onClick={() => props.onToggleColumn("source", true)}>column on</button>
      <button type="button" data-testid="column-off" onClick={() => props.onToggleColumn("source", false)}>column off</button>
      <button type="button" data-testid="clear" onClick={props.onClearFacets}>clear</button>
      <button type="button" data-testid="pause" onClick={props.onTogglePause}>pause</button>
      <button type="button" data-testid="export" disabled={!props.canExport} onClick={props.onExport}>export</button>
      <button type="button" data-testid="force-export" onClick={props.onExport}>force export</button>
      <button type="button" data-testid="close-filtered" onClick={props.onCloseFiltered}>close filtered</button>
    </div>
  ),
}))

vi.mock("@/features/observability/connection-list-table", () => ({
  ConnectionListTable: ({ onClose, onEmptyAction }: {
    onClose: (id: string) => void
    onEmptyAction?: () => void
  }) => (
    <div>
      <button type="button" data-testid="row-close" onClick={() => onClose("1")}>row close</button>
      {onEmptyAction ? <button type="button" data-testid="list-empty" onClick={onEmptyAction}>list empty</button> : null}
    </div>
  ),
}))

vi.mock("@/features/observability/connection-group-table", () => ({
  ConnectionGroupTable: ({ field, onCloseGroup, onEmptyAction }: {
    field: "outbound" | "rule" | "process"
    onCloseGroup: (field: "outbound" | "rule" | "process", key: string) => void
    onEmptyAction?: () => void
  }) => (
    <div>
      <button type="button" data-testid={`group-close-${field}`} onClick={() => onCloseGroup(field, "proxy")}>group close</button>
      {onEmptyAction ? <button type="button" data-testid={`group-empty-${field}`} onClick={onEmptyAction}>group empty</button> : null}
    </div>
  ),
}))

vi.mock("@/features/observability/connection-facet-summary", () => ({
  ConnectionFacetSummaryBar: ({ onChange }: { onChange: (patch: { protocol: string }) => void }) => (
    <button type="button" data-testid="facet-patch" onClick={() => onChange({ protocol: "tls" })}>facet</button>
  ),
}))

vi.mock("@/features/observability/stream-error-alert", () => ({
  StreamErrorAlert: ({ onReconnect }: { onReconnect: () => void }) => (
    <button type="button" data-testid="reconnect" onClick={onReconnect}>reconnect</button>
  ),
}))

vi.mock("@/features/observability/stream-status-badge", () => ({
  StreamStatusBadge: () => <span data-testid="stream-status" />,
}))

vi.mock("@/components/ui/tabs", () => ({
  Tabs: ({ children, onValueChange }: { children: ReactNode; onValueChange: (value: string) => void }) => (
    <div>
      {children}
      <button type="button" data-testid="tab-list" onClick={() => onValueChange("list")}>list</button>
      <button type="button" data-testid="tab-outbound" onClick={() => onValueChange("outbound")}>outbound</button>
      <button type="button" data-testid="tab-rule" onClick={() => onValueChange("rule")}>rule</button>
      <button type="button" data-testid="tab-process" onClick={() => onValueChange("process")}>process</button>
    </div>
  ),
  TabsContent: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  TabsList: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  TabsTrigger: ({ children }: { children: ReactNode }) => <span>{children}</span>,
}))

function renderPage(route = "/observability/connections") {
  return render(
    <I18nextProvider i18n={i18n}>
      <MemoryRouter initialEntries={[route]}>
        <ConnectionsPage />
      </MemoryRouter>
    </I18nextProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  state.stream.items = [{ list: state.connections }]
  state.stream.error = null
  state.stream.paused = false
  state.close.connections = state.connections
  state.close.closingId = null
  state.close.bulkBusy = false
  state.close.closeOne.mockResolvedValue(undefined)
  state.close.closeAll.mockResolvedValue(undefined)
  state.close.closeGroup.mockResolvedValue(undefined)
  state.close.closeFiltered.mockResolvedValue(undefined)
  state.downloadTextFile.mockReset()
  localStorage.clear()
})

describe("ConnectionsPage callbacks", () => {
  it("routes toolbar, tabs, row, group, and export callbacks", async () => {
    const user = userEvent.setup()
    renderPage()

    await user.click(screen.getByTestId("query"))
    await user.click(screen.getByTestId("empty-filters"))
    await user.click(screen.getByTestId("network"))
    await user.click(screen.getByTestId("protocol"))
    await user.click(screen.getByTestId("inbound"))
    await user.click(screen.getByTestId("outbound"))
    await user.click(screen.getByTestId("rule"))
    await user.click(screen.getByTestId("process"))
    await user.click(screen.getByTestId("sort-target"))
    await user.click(screen.getByTestId("sort-traffic"))
    await user.click(screen.getByTestId("column-on"))
    await user.click(screen.getByTestId("column-off"))
    await user.click(screen.getByTestId("pause"))
    await user.click(screen.getByTestId("facet-patch"))
    await user.click(screen.getByTestId("clear"))
    await user.click(screen.getByTestId("row-close"))
    await user.click(screen.getByTestId("group-close-outbound"))
    await user.click(screen.getByTestId("group-close-rule"))
    await user.click(screen.getByTestId("group-close-process"))
    await user.click(screen.getByTestId("close-filtered"))
    await user.click(screen.getByTestId("confirm-action"))
    await user.click(screen.getByTestId("export"))
    await user.click(screen.getByTestId("tab-list"))
    await user.click(screen.getByTestId("tab-outbound"))
    await user.click(screen.getByTestId("tab-rule"))
    await user.click(screen.getByTestId("tab-process"))

    expect(state.stream.setPaused).toHaveBeenCalledWith(true)
    expect(state.close.closeOne).toHaveBeenCalledWith("1")
    expect(state.close.closeAll).toHaveBeenCalled()
    expect(state.close.closeFiltered).toHaveBeenCalledWith(expect.any(Array))
    expect(state.close.closeGroup).toHaveBeenCalledWith("outbound", "proxy", expect.any(Array))
    expect(state.close.closeGroup).toHaveBeenCalledWith("rule", "proxy", expect.any(Array))
    expect(state.close.closeGroup).toHaveBeenCalledWith("process", "proxy", expect.any(Array))
    expect(state.downloadTextFile).toHaveBeenCalled()
    expect(state.toastSuccess).toHaveBeenCalled()
    expect(localStorage.getItem("boxd.connection-columns.v1")).toContain("target")
  })

  it("reports export failures and reconnects a failed stream", async () => {
    state.stream.error = new Error("stream disconnected")
    state.downloadTextFile.mockImplementation(() => {
      throw new Error("download failed")
    })
    const user = userEvent.setup()
    renderPage()
    await user.click(screen.getByTestId("reconnect"))
    await user.click(screen.getByTestId("export"))
    expect(state.stream.reconnect).toHaveBeenCalled()
    expect(state.toastError).toHaveBeenCalled()
  })

  it("renders the empty state and guards empty exports", async () => {
    state.stream.items = [{ list: [] }]
    state.close.connections = []
    const user = userEvent.setup()
    renderPage()
    expect(screen.getByText("暂无活跃连接")).toBeInTheDocument()
    expect(screen.getByTestId("export")).toBeDisabled()
    await user.click(screen.getByTestId("force-export"))
    expect(state.downloadTextFile).not.toHaveBeenCalled()
  })
})
