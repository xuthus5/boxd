import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"
import { toast } from "sonner"

import { useConfigRestore } from "@/features/dashboard/use-config-restore"
import type { ConfigApplyEvent } from "@/lib/api/types"
import { renderApp } from "@/test/render"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

const event: ConfigApplyEvent = {
  id: "event-1",
  source: "update",
  status: "applied",
  hash: "abcdef",
  size: 10,
  applied_at: "2026-07-23T00:00:00Z",
}

function RestoreHarness({ selected = event, onResult }: {
  selected?: ConfigApplyEvent
  onResult: (result: boolean) => void
}) {
  const { restore } = useConfigRestore()
  return <button type="button" onClick={() => { void restore(selected).then(onResult) }}>restore</button>
}

function renderHarness(onResult = vi.fn(), selected = event) {
  return renderApp(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <RestoreHarness selected={selected} onResult={onResult} />
    </QueryClientProvider>,
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.clearAllMocks()
})

describe("useConfigRestore", () => {
  it("reports when the selected snapshot is already current", async () => {
    const onResult = vi.fn()
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response(JSON.stringify({
      status: "ok",
      data: { restored: false, source_id: "event-1", already_current: true },
      error: null,
      meta: null,
    })))))
    renderHarness(onResult)
    await userEvent.setup().click(screen.getByRole("button", { name: "restore" }))
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith("该历史快照已是当前配置"))
    expect(onResult).toHaveBeenCalledWith(true)
  })

  it("guards concurrent restore requests", async () => {
    const onResult = vi.fn()
    let resolveRequest: ((response: Response) => void) | undefined
    const fetchMock = vi.fn(() => new Promise<Response>((resolve) => { resolveRequest = resolve }))
    vi.stubGlobal("fetch", fetchMock)
    renderHarness(onResult)
    const button = screen.getByRole("button", { name: "restore" })
    button.click()
    button.click()
    expect(fetchMock).toHaveBeenCalledTimes(1)
    await waitFor(() => expect(onResult).toHaveBeenCalledWith(false))
    resolveRequest?.(new Response(JSON.stringify({
      status: "ok", data: { restored: true, source_id: "event-1" }, error: null, meta: null,
    })))
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith("历史配置已恢复"))
    expect(onResult).toHaveBeenCalledWith(true)
  })

  it("returns false when the backend rolls the restore back", async () => {
    const onResult = vi.fn()
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response(JSON.stringify({
      status: "rolled_back",
      data: { restored: false, source_id: "event-1" },
      error: { code: "config_restart_failed", message: "restart failed" },
      meta: { rolled_back: true },
    })))))
    renderHarness(onResult)
    await userEvent.setup().click(screen.getByRole("button", { name: "restore" }))
    await waitFor(() => expect(onResult).toHaveBeenCalledWith(false))
    expect(toast.error).toHaveBeenCalled()
  })

  it("returns false when the restore request fails", async () => {
    const onResult = vi.fn()
    vi.stubGlobal("fetch", vi.fn(() => Promise.reject(new Error("network offline"))))
    renderHarness(onResult)
    await userEvent.setup().click(screen.getByRole("button", { name: "restore" }))
    await waitFor(() => expect(onResult).toHaveBeenCalledWith(false))
    expect(toast.error).toHaveBeenCalled()
  })

  it("returns false without requesting when the event has no id", async () => {
    const onResult = vi.fn()
    const fetchMock = vi.fn()
    vi.stubGlobal("fetch", fetchMock)
    renderHarness(onResult, { ...event, id: undefined })
    await userEvent.setup().click(screen.getByRole("button", { name: "restore" }))
    await waitFor(() => expect(onResult).toHaveBeenCalledWith(false))
    expect(fetchMock).not.toHaveBeenCalled()
  })
})
