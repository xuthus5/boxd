import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { expect, it, vi } from "vitest"

import { RouteRuleDialog } from "@/features/policy/route-rule-dialog"
import { renderApp } from "@/test/render"

it("keeps network type matcher when changing away from direct", async () => {
  const onSave = vi.fn()
  const user = userEvent.setup()
  renderApp(<RouteRuleDialog open title="编辑规则" item={{
    action: "direct", network_type: ["wifi"], bind_interface: "eth0",
  }} onOpenChange={vi.fn()} onSave={onSave} />)

  await user.click(screen.getByRole("tab", { name: "执行动作" }))
  await user.click(screen.getByRole("combobox", { name: "执行动作" }))
  await user.click(await screen.findByRole("option", { name: "reject" }))
  await user.click(screen.getByRole("button", { name: "保存" }))

  expect(onSave).toHaveBeenCalledWith({ action: "reject", network_type: ["wifi"] })
})
