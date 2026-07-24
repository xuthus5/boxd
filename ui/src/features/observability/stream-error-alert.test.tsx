import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

import { StreamErrorAlert } from "@/features/observability/stream-error-alert"
import { renderApp } from "@/test/render"

describe("StreamErrorAlert", () => {
  it("renders code, hint, copy, and reconnect control", async () => {
    const user = userEvent.setup()
    const onReconnect = vi.fn()
    renderApp(
      <StreamErrorAlert
        error="failed to fetch stream"
        path="/api/stats/logs"
        status="reconnecting"
        onReconnect={onReconnect}
      />,
    )
    expect(screen.getByTestId("stream-error-alert")).toHaveAttribute("data-stream-error-code", "network")
    expect(screen.getByText("failed to fetch stream")).toBeInTheDocument()
    expect(screen.getByText(/网络中断|Network interrupted/i)).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: /复制流错误|Copy stream error/i }))
    await user.click(screen.getByRole("button", { name: /立即重连|Reconnect now/i }))
    expect(onReconnect).toHaveBeenCalledTimes(1)
  })

  it("does not offer reconnect for an unauthorized stream", () => {
    renderApp(
      <StreamErrorAlert
        error="SSE request failed with status 401"
        status="closed"
        onReconnect={vi.fn()}
      />,
    )
    expect(screen.queryByRole("button", { name: /立即重连|Reconnect now/i })).toBeNull()
  })
})
