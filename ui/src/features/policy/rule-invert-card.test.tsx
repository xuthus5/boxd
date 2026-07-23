import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"
import { MemoryRouter } from "react-router-dom"

import { RouteRuleCard } from "@/features/policy/route-rule-card"
import { i18n } from "@/i18n"

describe("RouteRuleCard invert", () => {
  it("toggles invert from the quick switch", async () => {
    const onToggleInvert = vi.fn()
    const user = userEvent.setup()
    render(
      <I18nextProvider i18n={i18n}>
        <MemoryRouter>
          <RouteRuleCard
            index={0}
            item={{ action: "route", outbound: "proxy", domain: ["a.com"], invert: true }}
            first
            last
            onEdit={() => undefined}
            onCopy={() => undefined}
            onMoveUp={() => undefined}
            onMoveDown={() => undefined}
            onDelete={() => undefined}
            onToggleInvert={onToggleInvert}
          />
        </MemoryRouter>
      </I18nextProvider>,
    )
    expect(screen.getByText("已取反")).toBeInTheDocument()
    await user.click(screen.getByRole("switch", { name: "取反规则 1" }))
    expect(onToggleInvert).toHaveBeenCalled()
    expect(screen.getByRole("link", { name: "查看日志: 1" })).toHaveAttribute(
      "href",
      "/observability/logs?preset=route",
    )
    expect(screen.getByRole("link", { name: "查看连接: 1" })).toHaveAttribute(
      "href",
      "/observability/connections?outbound=proxy",
    )
  })
})
