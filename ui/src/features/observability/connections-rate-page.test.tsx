import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import { ConnectionsPage } from "@/features/observability/connections-page"
import { renderApp } from "@/test/render"

const state = vi.hoisted(() => {
  const snapshot = [{
    id: 1,
    target: "api.example.com:443",
    outbound: "proxy",
    upload: 250,
    download: 500,
    start: "2026-07-26T00:00:00Z",
  }]
  return {
    snapshot,
    rated: [{ ...snapshot[0], uploadRate: 150, downloadRate: 300 }],
    useRates: vi.fn(),
  }
})

vi.mock("@/features/auth/auth-context", () => ({
  useAuth: () => ({ session: { token: "token" } }),
}))

vi.mock("@/features/observability/use-stream-buffer", () => ({
  useStreamBuffer: () => ({
    items: [{ active_connections: 1, list: state.snapshot }],
    error: "",
    status: "open",
    paused: false,
    setPaused: vi.fn(),
    reconnect: vi.fn(),
  }),
}))

vi.mock("@/features/observability/use-connection-rates", () => ({
  useConnectionRates: (connections: unknown) => {
    state.useRates(connections)
    return state.rated
  },
}))

afterEach(() => {
  vi.restoreAllMocks()
  localStorage.clear()
})

describe("ConnectionsPage rates", () => {
  it("renders derived rates and exposes rate sorting", async () => {
    const user = userEvent.setup()
    localStorage.clear()
    renderApp(<ConnectionsPage />, "/observability/connections")

    expect(state.useRates).toHaveBeenCalledWith(state.snapshot)
    expect(screen.getByText("api.example.com:443")).toBeInTheDocument()
    expect(screen.getAllByText("↑ 150 B/s · ↓ 300 B/s").length).toBeGreaterThan(0)
    await user.click(screen.getByRole("combobox", { name: "排序连接" }))
    expect(await screen.findByRole("option", { name: "按实时速率" })).toBeInTheDocument()
  })

  it("renders aggregated rates in grouped views", () => {
    renderApp(<ConnectionsPage />, "/observability/connections?view=outbound")

    expect(screen.getByRole("row", {
      name: /proxy 1 250 B 500 B ↑ 150 B\/s · ↓ 300 B\/s/,
    })).toBeInTheDocument()
  })
})
