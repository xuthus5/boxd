import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"

import { RuleSetHealthFooter } from "@/features/dashboard/rule-set-health-footer"
import { renderApp } from "@/test/render"

describe("RuleSetHealthFooter", () => {
  it("shows pending state and navigation links", () => {
    renderApp(
      <RuleSetHealthFooter updatable={2} updateAction={{ isPending: true, mutate: vi.fn() }} />,
      "/dashboard",
    )

    const button = screen.getByRole("button", { name: "正在更新…" })
    expect(button).toBeDisabled()
    expect(button.querySelector("svg")).toHaveClass("animate-spin")
    expect(screen.getByRole("link", { name: "查看路由配置" })).toHaveAttribute("href", "/policy/route")
    expect(screen.getByRole("link", { name: "查看应用日志" })).toHaveAttribute("href", "/observability/logs?tab=application")
  })

  it("runs updates and disables the action without updatable entries", async () => {
    const mutate = vi.fn()
    const view = renderApp(
      <RuleSetHealthFooter updatable={1} updateAction={{ isPending: false, mutate }} />,
      "/dashboard",
    )
    await userEvent.click(screen.getByRole("button", { name: "更新可更新规则集" }))
    expect(mutate).toHaveBeenCalledOnce()

    view.unmount()
    renderApp(
      <RuleSetHealthFooter updatable={0} updateAction={{ isPending: false, mutate }} />,
      "/dashboard",
    )
    expect(screen.getByRole("button", { name: "更新可更新规则集" })).toBeDisabled()
  })
})
