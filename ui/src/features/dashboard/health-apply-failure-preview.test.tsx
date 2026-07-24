import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

import { HealthApplyFailurePreview } from "@/features/dashboard/health-apply-failure-preview"
import { renderApp } from "@/test/render"
import * as copy from "@/features/proxy/copy-tag-button"

describe("HealthApplyFailurePreview", () => {
  it("renders path, status, and copy control", async () => {
    const spy = vi.spyOn(copy, "copyText").mockResolvedValue()
    const user = userEvent.setup()
    renderApp(
      <HealthApplyFailurePreview
        count={1}
        event={{
          id: "1",
          source: "validate_inbounds",
          status: "validate_failed",
          hash: "deadbeef",
          size: 12,
          error: "inbounds[0].listen_port: invalid",
          error_code: "config_invalid",
          applied_at: "2026-07-24T12:00:00Z",
        }}
      />,
    )
    const root = document.querySelector('[data-slot="apply-failure-preview"]')
    expect(root).not.toBeNull()
    expect(screen.getByText(/1 次配置应用\/校验失败/)).toBeInTheDocument()
    expect(screen.getByText("inbounds[0].listen_port")).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "打开失败来源" })).toHaveAttribute(
      "href",
      "/advanced/raw?path=inbounds%5B0%5D.listen_port",
    )
    await user.click(screen.getByRole("button", { name: "复制错误" }))
    expect(spy).toHaveBeenCalled()
    expect(spy.mock.calls[0][0]).toContain("inbounds[0].listen_port")
  })

  it("renders nothing without failure", () => {
    const { container } = renderApp(<HealthApplyFailurePreview count={0} />)
    expect(container.querySelector('[data-slot="apply-failure-preview"]')).toBeNull()
  })
})
