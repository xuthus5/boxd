import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"

import { OutboundCard } from "@/features/proxy/outbound-card"
import { i18n } from "@/i18n"

function renderCard(item: Record<string, unknown>, onEdit = vi.fn(), onDelete = vi.fn()) {
  return {
    onEdit,
    onDelete,
    ...render(
      <I18nextProvider i18n={i18n}>
        <OutboundCard item={item as never} onEdit={onEdit} onDelete={onDelete} />
      </I18nextProvider>,
    ),
  }
}

describe("OutboundCard", () => {
  it("renders protocol outbound details with tls and transport badges", () => {
    renderCard({
      type: "vless",
      tag: "hk",
      server: "hk.example.com",
      server_port: 443,
      tls: { enabled: true },
      transport: { type: "ws" },
      detour: "direct",
    })
    expect(screen.getByRole("heading", { name: "hk" })).toBeInTheDocument()
    expect(screen.getByText("hk.example.com:443")).toBeInTheDocument()
    expect(screen.getByText("vless")).toBeInTheDocument()
    expect(screen.getByText("TLS")).toBeInTheDocument()
    expect(screen.getByText("ws")).toBeInTheDocument()
    expect(screen.getByText("前置出站：direct")).toBeInTheDocument()
  })

  it("renders group outbounds without server and with members", () => {
    renderCard({ type: "selector", tag: "proxy", outbounds: ["a", "b", 1, null], transport: "bad" })
    expect(screen.getByText("分组出站")).toBeInTheDocument()
    expect(screen.getByText("成员：a, b")).toBeInTheDocument()
    expect(screen.queryByText("TLS")).not.toBeInTheDocument()
  })

  it("falls back for missing tag/type and confirms delete", async () => {
    const { onEdit, onDelete } = renderCard({ server: 1, tls: { enabled: false }, transport: { type: null } })
    expect(screen.getByRole("heading", { name: "—" })).toBeInTheDocument()
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "编辑" }))
    expect(onEdit).toHaveBeenCalled()
    await user.click(screen.getByRole("button", { name: "删除" }))
    await user.click(screen.getByRole("button", { name: "确认删除" }))
    expect(onDelete).toHaveBeenCalled()
  })

})
