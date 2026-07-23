import type { ReactElement } from "react"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"

import { InboundCard } from "@/features/proxy/inbound-card"
import { i18n } from "@/i18n"

function wrap(ui: ReactElement) {
  return render(<I18nextProvider i18n={i18n}>{ui}</I18nextProvider>)
}

describe("InboundCard", () => {
  it("toggles system proxy for mixed inbound", async () => {
    const onPatch = vi.fn()
    const user = userEvent.setup()
    wrap(
      <InboundCard
        item={{ tag: "mixed-in", type: "mixed", listen: "::", listen_port: 1080, set_system_proxy: false }}
        onEdit={() => undefined}
        onDelete={() => undefined}
        onPatch={onPatch}
      />,
    )
    expect(screen.getByText("mixed")).toBeInTheDocument()
    await user.click(screen.getByRole("switch", { name: "设置系统代理" }))
    expect(onPatch).toHaveBeenCalledWith(expect.objectContaining({ set_system_proxy: true }))
  })

  it("toggles auto_route for tun inbound", async () => {
    const onPatch = vi.fn()
    const user = userEvent.setup()
    wrap(
      <InboundCard
        item={{ tag: "tun-in", type: "tun", interface_name: "boxd0", auto_route: true }}
        onEdit={() => undefined}
        onDelete={() => undefined}
        onPatch={onPatch}
      />,
    )
    expect(screen.getByText("已自动路由")).toBeInTheDocument()
    const toggle = screen.getByRole("switch", { name: "自动路由" })
    expect(toggle).toBeChecked()
    await user.click(toggle)
    expect(onPatch).toHaveBeenCalledWith(expect.objectContaining({ auto_route: false }))
  })

  it("hides quick toggles without onPatch", () => {
    wrap(
      <InboundCard
        item={{ tag: "mixed-in", type: "mixed", listen: "::", listen_port: 1080, set_system_proxy: true }}
        onEdit={() => undefined}
        onDelete={() => undefined}
      />,
    )
    expect(screen.getByText("系统代理已开")).toBeInTheDocument()
    expect(screen.queryByRole("switch")).not.toBeInTheDocument()
  })

})
