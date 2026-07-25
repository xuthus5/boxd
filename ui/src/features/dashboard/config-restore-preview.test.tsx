import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"
import { toast } from "sonner"

import { ConfigApplyTimelineCard } from "@/features/dashboard/config-apply-timeline-card"
import type { SingBoxConfig } from "@/lib/api/types"
import { renderApp } from "@/test/render"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

const currentConfig: SingBoxConfig = {
  log: { level: "info" },
  outbounds: [{ type: "direct", tag: "direct" }],
}
const historicalConfig: SingBoxConfig = {
  log: { level: "debug" },
  outbounds: [{ type: "block", tag: "block" }],
}
const event = {
  id: "event/2",
  source: "update",
  status: "applied",
  hash: "1111222233334444",
  size: 512,
  restorable: true,
  applied_at: "2026-07-23T11:00:00Z",
}

type MockOptions = {
  current?: SingBoxConfig
  snapshot?: SingBoxConfig
  currentResponse?: Promise<Response>
  restoreResponse?: Promise<Response>
  currentError?: boolean
  snapshotError?: boolean
  restoreStatus?: "ok" | "rolled_back"
}

function response(body: unknown, status = 200) {
  return Promise.resolve(new Response(JSON.stringify(body), { status }))
}

function requestPath(input: string | URL | Request) {
  if (typeof input === "string") return input
  if (input instanceof URL) return input.pathname
  return new URL(input.url).pathname
}

function mockAPI(options: MockOptions = {}) {
  return vi.fn((input: string | URL | Request) => {
    const path = requestPath(input)
    if (path === "/api/config/") {
      if (options.currentResponse) return options.currentResponse
      if (options.currentError) {
        return response({ status: "error", data: null, error: { code: "internal_error", message: "config unavailable" }, meta: null }, 500)
      }
      return response(options.current ?? currentConfig)
    }
    if (path === "/api/config/apply-history") return response({ events: [event] })
    if (path.endsWith("/snapshot")) {
      if (options.snapshotError) {
        return response({ status: "error", data: null, error: { code: "internal_error", message: "snapshot unavailable" }, meta: null }, 500)
      }
      return response(options.snapshot ?? historicalConfig)
    }
    if (path.endsWith("/restore") && options.restoreResponse) return options.restoreResponse
    if (path.endsWith("/restore") && options.restoreStatus === "rolled_back") {
      return response({
        status: "rolled_back",
        data: { restored: false, source_id: event.id },
        error: { code: "config_restart_failed", message: "restart failed" },
        meta: { rolled_back: true },
      })
    }
    if (path.endsWith("/restore")) {
      return response({ status: "ok", data: { restored: true, source_id: event.id }, error: null, meta: null })
    }
    return response({})
  })
}

function renderCard(fetchMock: ReturnType<typeof mockAPI>) {
  vi.stubGlobal("fetch", fetchMock)
  return renderApp(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <ConfigApplyTimelineCard />
    </QueryClientProvider>,
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.clearAllMocks()
})

describe("config restore preview", () => {
  it("previews path-level changes before restoring", async () => {
    const fetchMock = mockAPI()
    renderCard(fetchMock)
    const user = userEvent.setup()
    await user.click(await screen.findByRole("button", { name: "恢复此配置" }))
    expect(await screen.findByText("log.level")).toBeInTheDocument()
    expect(screen.getByText('"info"')).toBeInTheDocument()
    expect(screen.getByText('"debug"')).toBeInTheDocument()
    const confirm = screen.getByRole("button", { name: "确认恢复" })
    expect(confirm).toBeEnabled()
    await user.click(confirm)
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(
      "/api/config/apply-history/event%2F2/restore",
      expect.objectContaining({ method: "POST" }),
    ))
    await waitFor(() => expect(screen.queryByText("恢复历史配置？")).not.toBeInTheDocument())
    expect(toast.success).toHaveBeenCalledWith("历史配置已恢复")
  })

  it("disables restore when the snapshot has no changes", async () => {
    renderCard(mockAPI({ snapshot: currentConfig }))
    const user = userEvent.setup()
    await user.click(await screen.findByRole("button", { name: "恢复此配置" }))
    expect(await screen.findByText("配置无变化")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "确认恢复" })).toBeDisabled()
    await user.click(screen.getByRole("button", { name: "取消" }))
    await waitFor(() => expect(screen.queryByText("恢复历史配置？")).not.toBeInTheDocument())
  })

  it("shows snapshot diagnostics and keeps restore disabled", async () => {
    const fetchMock = mockAPI({ snapshotError: true })
    renderCard(fetchMock)
    const user = userEvent.setup()
    await user.click(await screen.findByRole("button", { name: "恢复此配置" }))
    const error = await screen.findByTestId("card-query-error")
    expect(error).toHaveAttribute("data-error-code", "internal")
    expect(screen.getByText("snapshot unavailable")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "重试" }))
    await waitFor(() => expect(fetchMock.mock.calls.filter(([input]) => requestPath(input).endsWith("/snapshot"))).toHaveLength(2))
    expect(screen.getByRole("button", { name: "确认恢复" })).toBeDisabled()
  })

  it("waits for the current config before enabling comparison", async () => {
    let resolveCurrent: ((value: Response) => void) | undefined
    const currentResponse = new Promise<Response>((resolve) => { resolveCurrent = resolve })
    renderCard(mockAPI({ currentResponse }))
    await userEvent.setup().click(await screen.findByRole("button", { name: "恢复此配置" }))
    expect(await screen.findByText("正在读取当前配置…")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "确认恢复" })).toBeDisabled()
    resolveCurrent?.(new Response(JSON.stringify(currentConfig)))
    expect(await screen.findByText("log.level")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "确认恢复" })).toBeEnabled()
  })

  it("shows restore progress and prevents closing until success", async () => {
    let resolveRestore: ((value: Response) => void) | undefined
    const restoreResponse = new Promise<Response>((resolve) => { resolveRestore = resolve })
    renderCard(mockAPI({ restoreResponse }))
    const user = userEvent.setup()
    await user.click(await screen.findByRole("button", { name: "恢复此配置" }))
    const confirm = await screen.findByRole("button", { name: "确认恢复" })
    await waitFor(() => expect(confirm).toBeEnabled())
    await user.click(confirm)
    const restoringButtons = await screen.findAllByRole("button", { name: /恢复中/ })
    restoringButtons.forEach((button) => expect(button).toBeDisabled())
    expect(screen.getByRole("button", { name: "取消" })).toBeDisabled()
    await user.keyboard("{Escape}")
    expect(screen.getByText("恢复历史配置？")).toBeInTheDocument()
    resolveRestore?.(new Response(JSON.stringify({
      status: "ok", data: { restored: true, source_id: event.id }, error: null, meta: null,
    })))
    await waitFor(() => expect(screen.queryByText("恢复历史配置？")).not.toBeInTheDocument())
  })

  it("keeps the dialog open when restore rolls back", async () => {
    renderCard(mockAPI({ restoreStatus: "rolled_back" }))
    const user = userEvent.setup()
    await user.click(await screen.findByRole("button", { name: "恢复此配置" }))
    await user.click(await screen.findByRole("button", { name: "确认恢复" }))
    await waitFor(() => expect(toast.error).toHaveBeenCalled())
    expect(screen.getByText("恢复历史配置？")).toBeInTheDocument()
  })

  it("disables restore when the current config is unavailable", async () => {
    renderCard(mockAPI({ currentError: true }))
    await userEvent.setup().click(await screen.findByRole("button", { name: "恢复此配置" }))
    expect(await screen.findByText("无法读取当前配置，暂时不能比较或恢复。")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "确认恢复" })).toBeDisabled()
  })
})
