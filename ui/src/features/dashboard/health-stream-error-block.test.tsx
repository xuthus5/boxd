import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

import { HealthStreamErrorBlock } from "@/features/dashboard/health-stream-error-block"
import { renderApp } from "@/test/render"
import * as copy from "@/features/proxy/copy-tag-button"

describe("HealthStreamErrorBlock", () => {
  it("renders densified stream error with copy", async () => {
    const spy = vi.spyOn(copy, "copyText").mockResolvedValue()
    const user = userEvent.setup()
    renderApp(
      <HealthStreamErrorBlock
        error="failed to fetch"
        status="error"
        path="/api/stats/connections"
      />,
    )
    expect(screen.getByText("failed to fetch")).toBeInTheDocument()
    expect(document.querySelector('[data-slot="health-stream-error"]')).toHaveAttribute(
      "data-error-code",
      "network",
    )
    await user.click(screen.getByRole("button", { name: "复制流错误" }))
    expect(spy).toHaveBeenCalled()
    expect(spy.mock.calls[0][0]).toContain("failed to fetch")
  })

  it("renders nothing when healthy", () => {
    const { container } = renderApp(<HealthStreamErrorBlock />)
    expect(container.querySelector('[data-slot="health-stream-error"]')).toBeNull()
  })

  it("renders optional action link", () => {
    renderApp(
      <HealthStreamErrorBlock
        error="failed to fetch"
        status="error"
        href="/observability/connections"
      />,
    )
    expect(screen.getByRole("link", { name: "查看连接" })).toHaveAttribute(
      "href",
      "/observability/connections",
    )
  })
})
