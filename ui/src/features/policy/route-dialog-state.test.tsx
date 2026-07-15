import { fireEvent, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"

import { RouteRuleDialog } from "@/features/policy/route-rule-dialog"
import { RouteRuleSetDialog } from "@/features/policy/route-rule-set-dialog"
import { renderApp } from "@/test/render"

async function replaceJSON(label: string, value: string) {
  const editor = screen.getByRole("textbox", { name: label })
  await userEvent.click(editor)
  await userEvent.keyboard("{Control>}a{/Control}")
  await userEvent.paste(value)
}

const dialogs = [
  {
    name: "rule",
    dialog: <RouteRuleDialog open title="编辑规则" item={{ action: "reject", domain: ["old.example"] }}
      onOpenChange={vi.fn()} onSave={vi.fn()} />,
    jsonLabel: "编辑规则 JSON",
    tab: "域名与地址",
    field: "域名",
    recovered: "new.example",
  },
  {
    name: "rule set",
    dialog: <RouteRuleSetDialog open title="编辑规则集" item={{ type: "inline", tag: "old" }}
      onOpenChange={vi.fn()} onSave={vi.fn()} />,
    jsonLabel: "编辑规则集 JSON",
    tab: "基础",
    field: "Tag",
    recovered: "new",
  },
] as const

describe("Route Advanced JSON resilience", () => {
  it.each(dialogs)("keeps $name Tabs and invalid raw JSON mounted", async ({ dialog, jsonLabel, tab, field }) => {
    renderApp(dialog)
    await userEvent.click(screen.getByRole("tab", { name: "高级 JSON" }))
    await replaceJSON(jsonLabel, "[")

    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    expect(screen.getByRole("tab", { name: tab })).toBeInTheDocument()
    await userEvent.click(screen.getByRole("tab", { name: tab }))
    expect(screen.getByLabelText(field)).toBeInTheDocument()
    await userEvent.click(screen.getByRole("tab", { name: "高级 JSON" }))
    expect(screen.getByRole("textbox", { name: jsonLabel })).toHaveTextContent("[")
  })

  it.each(dialogs)("recovers $name JSON through a structured edit", async ({ dialog, jsonLabel, tab, field, recovered }) => {
    renderApp(dialog)
    await userEvent.click(screen.getByRole("tab", { name: "高级 JSON" }))
    await replaceJSON(jsonLabel, "[")
    await userEvent.click(screen.getByRole("tab", { name: tab }))
    fireEvent.change(screen.getByLabelText(field), { target: { value: recovered } })

    expect(screen.getByRole("button", { name: "保存" })).toBeEnabled()
    await userEvent.click(screen.getByRole("tab", { name: "高级 JSON" }))
    expect(screen.getByRole("textbox", { name: jsonLabel })).toHaveTextContent(`"${recovered}"`)
  })
})
