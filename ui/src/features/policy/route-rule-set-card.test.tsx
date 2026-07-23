import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"

import { RouteRuleSetCard } from "@/features/policy/route-rule-set-card"
import { i18n } from "@/i18n"

function renderCard(options: {
  item?: Record<string, unknown>
  status?: Parameters<typeof RouteRuleSetCard>[0]["status"]
  lastUpdate?: Parameters<typeof RouteRuleSetCard>[0]["lastUpdate"]
  updating?: boolean
  withUpdate?: boolean
} = {}) {
  const onEdit = vi.fn()
  const onCopy = vi.fn()
  const onDelete = vi.fn()
  const onUpdate = vi.fn()
  render(
    <I18nextProvider i18n={i18n}>
      <RouteRuleSetCard
        item={(options.item ?? { type: "remote", tag: "geo", url: "https://example.com/geo.srs" }) as never}
        status={options.status}
        lastUpdate={options.lastUpdate}
        updating={options.updating}
        onEdit={onEdit}
        onCopy={onCopy}
        onDelete={onDelete}
        onUpdate={options.withUpdate === false ? undefined : onUpdate}
      />
    </I18nextProvider>,
  )
  return { onEdit, onCopy, onDelete, onUpdate }
}

describe("RouteRuleSetCard", () => {
  it("shows update controls for updatable rule sets", async () => {
    const { onUpdate } = renderCard({
      status: {
        tag: "geo",
        type: "remote",
        updatable: true,
        builtin: true,
        update_interval: "24h",
        last_updated: "2026-01-02T03:04:05Z",
        note: "ready",
      },
    })
    expect(screen.getByText("内置")).toBeInTheDocument()
    expect(screen.getByText("间隔 24h")).toBeInTheDocument()
    expect(screen.getByText(/上次更新/)).toBeInTheDocument()
    expect(screen.getByText("ready")).toBeInTheDocument()
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "更新规则集 geo" }))
    expect(onUpdate).toHaveBeenCalled()
  })

  it("hides update button when not updatable and formats invalid dates", () => {
    renderCard({
      item: { type: "inline", rules: [] },
      status: { tag: "", type: "inline", builtin: false, updatable: false, last_updated: "not-a-date" },
      withUpdate: false,
    })
    expect(screen.getByText("未命名")).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /更新规则集/ })).not.toBeInTheDocument()
    expect(screen.getByText(/上次更新：not-a-date/)).toBeInTheDocument()
  })

  it("confirms delete from desktop actions", async () => {
    const { onDelete, onCopy, onEdit } = renderCard()
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "编辑规则集 geo" }))
    expect(onEdit).toHaveBeenCalled()
    await user.click(screen.getByRole("button", { name: "复制规则集 geo" }))
    expect(onCopy).toHaveBeenCalled()
    await user.click(screen.getByRole("button", { name: "删除规则集 geo" }))
    await user.click(screen.getByRole("button", { name: "确认删除" }))
    expect(onDelete).toHaveBeenCalled()
  })
})


describe("RouteRuleSetCard update errors", () => {
  it("renders failed update diagnostics and copies them", async () => {
    const copy = await import("@/features/proxy/copy-tag-button")
    const spy = vi.spyOn(copy, "copyText").mockResolvedValue()
    renderCard({
      status: { tag: "geo", type: "remote", updatable: true, builtin: false },
      lastUpdate: {
        tag: "geo",
        type: "remote",
        ok: false,
        error: "unexpected status 500",
        error_code: "http_status",
      },
    })
    expect(screen.getByText("http_status")).toBeInTheDocument()
    expect(screen.getByText("unexpected status 500")).toBeInTheDocument()
    expect(screen.getByText(/非成功状态码|non-success/i)).toBeInTheDocument()
    await userEvent.setup().click(screen.getByRole("button", { name: "复制更新错误: geo" }))
    expect(spy).toHaveBeenCalled()
    expect(String(spy.mock.calls[0][0])).toContain("code: http_status")
    spy.mockRestore()
  })
})
