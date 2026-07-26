import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { I18nextProvider } from "react-i18next"
import { describe, expect, it, vi } from "vitest"

import { ServiceCard } from "@/features/advanced/service-card"
import { i18n } from "@/i18n"

describe("ServiceCard", () => {
  it("renders listener and SSM server summaries", () => {
    render(
      <I18nextProvider i18n={i18n}>
        <ServiceCard
          item={{
            type: "ssm-api",
            tag: "manager",
            listen: "127.0.0.1",
            listen_port: 8080,
            servers: { "/": "ss-in", "/backup": "ss-backup" },
          }}
          onEdit={vi.fn()}
          onDelete={vi.fn()}
        />
      </I18nextProvider>,
    )
    expect(screen.getByRole("heading", { name: "manager" })).toBeInTheDocument()
    expect(screen.getByText("ssm-api")).toBeInTheDocument()
    expect(screen.getAllByText("127.0.0.1:8080").length).toBeGreaterThan(0)
    expect(screen.getByText("2 个 Shadowsocks 映射")).toBeInTheDocument()
  })

  it("falls back to an unnamed service and confirms deletion", async () => {
    const onEdit = vi.fn()
    const onDelete = vi.fn()
    render(
      <I18nextProvider i18n={i18n}>
        <ServiceCard item={{ type: "resolved" }} onEdit={onEdit} onDelete={onDelete} />
      </I18nextProvider>,
    )
    expect(screen.getByRole("heading", { name: "未命名服务" })).toBeInTheDocument()
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "编辑" }))
    expect(onEdit).toHaveBeenCalled()
    await user.click(screen.getByRole("button", { name: "删除" }))
    await user.click(screen.getByRole("button", { name: /确认/ }))
    expect(onDelete).toHaveBeenCalled()
  })
})
