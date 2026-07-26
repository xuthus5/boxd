import type { ReactElement } from "react"
import { cleanup, fireEvent, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { describe, expect, it, vi } from "vitest"

import { RouteRuleDialog } from "@/features/policy/route-rule-dialog"
import { RouteRuleSetDialog } from "@/features/policy/route-rule-set-dialog"
import type { JsonObject } from "@/features/policy/policy-form-model"
import { renderApp } from "@/test/render"

function renderDialog(ui: ReactElement) {
  return renderApp(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>{ui}</QueryClientProvider>)
}


async function choose(label: string, option: string) {
  const user = userEvent.setup()
  await user.click(screen.getByRole("combobox", { name: label }))
  await user.click(await screen.findByRole("option", { name: option }))
}

interface RuleSetRequiredCase {
  name: string
  item: JsonObject
  label: string
  value: string
  expected: JsonObject
}

const ruleSetRequiredCases: RuleSetRequiredCase[] = [
  {
    name: "remote Tag", item: { type: "remote", url: "https://example/r.srs" },
    label: "Tag", value: "remote", expected: { type: "remote", tag: "remote", url: "https://example/r.srs" },
  },
  {
    name: "remote URL", item: { type: "remote", tag: "remote" },
    label: "远程 URL", value: "https://example/r.srs", expected: { type: "remote", tag: "remote", url: "https://example/r.srs" },
  },
  {
    name: "local path", item: { type: "local", tag: "local" },
    label: "本地路径", value: "/etc/local.srs", expected: { type: "local", tag: "local", path: "/etc/local.srs" },
  },
  {
    name: "inline Tag", item: { type: "inline", rules: [] },
    label: "Tag", value: "inline", expected: { type: "inline", tag: "inline", rules: [] },
  },
]

function expectEveryFieldGrouped() {
  const fields = [...document.querySelectorAll('[data-slot="field"]')]
  expect(fields.length).toBeGreaterThan(0)
  expect(fields.every((field) => field.closest('[data-slot="field-group"]'))).toBe(true)
}

describe("route dialog required values", () => {
  it("requires a complete logical child after changing from default", async () => {
    renderDialog(<RouteRuleDialog open title="编辑规则" item={{ action: "reject" }}
      onOpenChange={vi.fn()} onSave={vi.fn()} />)

    await choose("规则类型", "logical")
    await choose("逻辑模式", "and")
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()

    await userEvent.click(screen.getAllByRole("button", { name: "新增子规则" })[0])
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    await userEvent.click(screen.getByRole("tab", { name: "执行动作" }))
    await choose("执行动作", "reject")
    await userEvent.click(screen.getByRole("button", { name: "保存" }))
    expect(screen.getByRole("button", { name: "保存" })).toBeEnabled()
  })

  it.each(ruleSetRequiredCases)("requires $name and saves after it is provided", async ({ item, label, value, expected }) => {
    const onSave = vi.fn()
    renderDialog(<RouteRuleSetDialog open title="编辑规则集" item={item}
      onOpenChange={vi.fn()} onSave={onSave} />)

    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    fireEvent.change(screen.getByLabelText(label), { target: { value } })
    expect(screen.getByRole("button", { name: "保存" })).toBeEnabled()
    await userEvent.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenCalledWith(expected)
  })

  it("edits direct logical children structurally and keeps nested rules as JSON", async () => {
    renderDialog(<RouteRuleDialog open title="编辑规则" item={{
      type: "logical", mode: "and", action: "reject",
      rules: [{ type: "logical", mode: "or", action: "reject", rules: [{ action: "reject" }] }],
    }} onOpenChange={vi.fn()} onSave={vi.fn()} />)

    expect(screen.getByRole("heading", { name: "逻辑子规则" })).toBeInTheDocument()
    expect(screen.queryByLabelText("子规则 JSON")).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole("button", { name: "编辑子规则 1" }))
    expect(screen.getByLabelText("子规则 JSON")).toBeInTheDocument()
  })
})

describe("unknown route values", () => {
  it("keeps an unknown rule-set type and payload when saving", async () => {
    const item = { type: "future", tag: "custom", format: "binary", payload: { enabled: true } }
    const onSave = vi.fn()
    const user = userEvent.setup()
    renderDialog(<RouteRuleSetDialog open title="编辑规则集" item={item}
      onOpenChange={vi.fn()} onSave={onSave} />)

    await user.click(screen.getByRole("combobox", { name: "规则集类型" }))
    await user.click(await screen.findByRole("option", { name: "future" }))
    await user.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenCalledWith(item)
  })

  it("switches an unknown rule type through the model transition", async () => {
    const onSave = vi.fn()
    renderDialog(<RouteRuleDialog open title="编辑规则" item={{
      type: "future", mode: "and", domain: ["example.com"], action: "reject", payload: { enabled: true },
    }} onOpenChange={vi.fn()} onSave={onSave} />)

    await userEvent.click(screen.getByRole("combobox", { name: "规则类型" }))
    expect(await screen.findByRole("option", { name: "future" })).toBeInTheDocument()
    await userEvent.click(screen.getByRole("option", { name: "default" }))
    await userEvent.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenCalledWith({
      domain: ["example.com"], action: "reject", payload: { enabled: true },
    }, { name: "", description: "" })
  })

  it("switches an unknown action and preserves its payload", async () => {
    const onSave = vi.fn()
    renderDialog(<RouteRuleDialog open title="编辑规则" item={{
      action: "future", method: "drop", outbound: "old", payload: { enabled: true },
    }} onOpenChange={vi.fn()} onSave={onSave} />)

    await userEvent.click(screen.getByRole("tab", { name: "执行动作" }))
    await userEvent.click(screen.getByRole("combobox", { name: "执行动作" }))
    expect(await screen.findByRole("option", { name: "future" })).toBeInTheDocument()
    await userEvent.click(screen.getByRole("option", { name: "reject" }))
    await userEvent.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenCalledWith({ action: "reject", method: "drop", payload: { enabled: true } }, { name: "", description: "" })
  })
})

describe("route dialog form composition", () => {
  it("wraps every direct Field in a FieldGroup", async () => {
    renderDialog(<RouteRuleDialog open title="编辑规则" item={{ action: "reject" }}
      onOpenChange={vi.fn()} onSave={vi.fn()} />)
    expectEveryFieldGrouped()
    await userEvent.click(screen.getByRole("tab", { name: "高级 JSON" }))
    expectEveryFieldGrouped()

    cleanup()
    renderDialog(<RouteRuleSetDialog open title="编辑规则集" item={{ type: "inline", tag: "inline" }}
      onOpenChange={vi.fn()} onSave={vi.fn()} />)
    expectEveryFieldGrouped()
    await userEvent.click(screen.getByRole("tab", { name: "高级 JSON" }))
    expectEveryFieldGrouped()
  })
})
