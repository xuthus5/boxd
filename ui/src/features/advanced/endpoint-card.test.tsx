import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"

import { EndpointCard } from "@/features/advanced/endpoint-card"
import { i18n } from "@/i18n"

describe("EndpointCard", () => {
  it("renders wireguard peers and detail badges", () => {
    render(
      <I18nextProvider i18n={i18n}>
        <EndpointCard
          item={{ type: "wireguard", tag: "wg", address: ["10.0.0.2/32"], peers: [{}, {}] }}
          onEdit={vi.fn()}
          onDelete={vi.fn()}
        />
      </I18nextProvider>,
    )
    expect(screen.getByRole("heading", { name: "wg" })).toBeInTheDocument()
    expect(screen.getByText("wireguard")).toBeInTheDocument()
    expect(screen.getAllByText("10.0.0.2/32").length).toBeGreaterThan(0)
  })

  it("falls back to unnamed endpoint and confirms delete", async () => {
    const onEdit = vi.fn()
    const onDelete = vi.fn()
    render(
      <I18nextProvider i18n={i18n}>
        <EndpointCard item={{ type: "tailscale" }} onEdit={onEdit} onDelete={onDelete} />
      </I18nextProvider>,
    )
    expect(screen.getByRole("heading", { name: "未命名端点" })).toBeInTheDocument()
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "编辑" }))
    expect(onEdit).toHaveBeenCalled()
    await user.click(screen.getByRole("button", { name: "删除" }))
    await user.click(screen.getByRole("button", { name: /确认/ }))
    expect(onDelete).toHaveBeenCalled()
  })
})
