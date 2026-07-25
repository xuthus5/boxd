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

function RestoreHarness() {
  const { restore } = useConfigRestore()
  return <button type="button" onClick={() => { void restore(event) }}>restore</button>
}

function renderHarness() {
  return renderApp(
    <QueryClientProvider client={new QueryClient()}>
      <RestoreHarness />
    </QueryClientProvider>,
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.clearAllMocks()
})

describe("useConfigRestore", () => {
  it("reports when the selected snapshot is already current", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response(JSON.stringify({
      status: "ok",
      data: { restored: false, source_id: "event-1", already_current: true },
      error: null,
      meta: null,
    })))))
    renderHarness()
    await userEvent.setup().click(screen.getByRole("button", { name: "restore" }))
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith("该历史快照已是当前配置"))
  })

  it("guards concurrent restore requests", async () => {
    let resolveRequest: ((response: Response) => void) | undefined
    const fetchMock = vi.fn(() => new Promise<Response>((resolve) => { resolveRequest = resolve }))
    vi.stubGlobal("fetch", fetchMock)
    renderHarness()
    const button = screen.getByRole("button", { name: "restore" })
    button.click()
    button.click()
    expect(fetchMock).toHaveBeenCalledTimes(1)
    resolveRequest?.(new Response(JSON.stringify({
      status: "ok", data: { restored: true, source_id: "event-1" }, error: null, meta: null,
    })))
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith("历史配置已恢复"))
  })
})
