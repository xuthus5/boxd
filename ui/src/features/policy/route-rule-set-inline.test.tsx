import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { fireEvent, screen, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { useState } from "react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { i18n } from "@/i18n"
import { RouteHeadlessRuleDialog } from "@/features/policy/route-headless-rule-dialog"
import { RouteInlineRuleSetEditor } from "@/features/policy/route-inline-rule-set-editor"
import { RouteRuleSetDialog } from "@/features/policy/route-rule-set-dialog"
import type { JsonObject } from "@/features/policy/policy-form-model"
import { installMockAPI } from "@/test/mock-api"
import { renderApp } from "@/test/render"

function renderDialog(ui: React.ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return renderApp(<QueryClientProvider client={client}>{ui}</QueryClientProvider>)
}

function InlineHarness({ initial }: { initial: JsonObject }) {
  const [item, setItem] = useState(initial)
  return <><RouteInlineRuleSetEditor item={item} onChange={setItem} />
    <output aria-label="inline state">{JSON.stringify(item)}</output></>
}

async function choose(dialog: HTMLElement, label: string, option: string) {
  const user = userEvent.setup()
  await user.click(within(dialog).getByRole("combobox", { name: label }))
  await user.click(await screen.findByRole("option", { name: option }))
}

beforeEach(async () => {
  installMockAPI()
  await i18n.changeLanguage("zh")
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe("inline rule-set editor", () => {
  it("adds a rule from an empty set and cancels a later edit", async () => {
    renderDialog(<InlineHarness initial={{ type: "inline", tag: "empty", rules: [] }} />)

    expect(screen.getByText("暂无 Inline 规则")).toBeInTheDocument()
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "新增 Inline 规则" }))
    const dialog = await screen.findByRole("dialog", { name: "新增 Inline 规则" })
    await user.click(within(dialog).getByRole("tab", { name: "域名与地址" }))
    fireEvent.change(within(dialog).getByLabelText("域名"), { target: { value: "example.com" } })
    await user.click(within(dialog).getByRole("button", { name: "保存" }))

    expect(JSON.parse(screen.getByLabelText("inline state").textContent ?? "{}").rules)
      .toEqual([{ domain: ["example.com"] }])
    await user.click(screen.getByRole("button", { name: "新增 Inline 规则" }))
    const secondDialog = await screen.findByRole("dialog", { name: "新增 Inline 规则" })
    await user.click(within(secondDialog).getByRole("button", { name: "取消" }))
    expect(screen.queryByRole("dialog", { name: "新增 Inline 规则" })).not.toBeInTheDocument()
  })

  it("renders logical summaries, empty summaries, overflow badges, and move boundaries", async () => {
    renderDialog(<InlineHarness initial={{
      type: "inline", tag: "summary", rules: [
        { domain: ["a.example"], domain_suffix: ["b.example"], domain_keyword: ["keyword"], domain_regex: [".*"], source_ip_cidr: ["192.0.2.0/24"], ip_cidr: ["198.51.100.0/24"] },
        { type: "logical", mode: "and", rules: [{ domain: ["a.example"] }, { domain_suffix: ["b.example"] }] },
        {},
      ],
    }} />)

    expect(screen.getByText("+1")).toBeInTheDocument()
    expect(screen.getByText("2 条子规则")).toBeInTheDocument()
    expect(screen.getByText("暂无可见匹配摘要")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "上移 Inline 规则 1" })).toBeDisabled()
    expect(screen.getByRole("button", { name: "下移 Inline 规则 3" })).toBeDisabled()

    await userEvent.setup().click(screen.getByRole("button", { name: "上移 Inline 规则 2" }))
    const rules = JSON.parse(screen.getByLabelText("inline state").textContent ?? "{}").rules
    expect(rules[0]).toEqual({ type: "logical", mode: "and", rules: [{ domain: ["a.example"] }, { domain_suffix: ["b.example"] }] })
  })

  it("edits a structured headless rule and saves it with the parent rule set", async () => {
    const onSave = vi.fn()
    renderDialog(<RouteRuleSetDialog open title="编辑规则集" item={{
      type: "inline", tag: "private", rules: [{ domain_suffix: ["example.com"] }],
    }} onOpenChange={vi.fn()} onSave={onSave} />)

    const parent = screen.getByRole("dialog", { name: "编辑规则集" })
    expect(within(parent).getByText("共 1 条 Inline 规则")).toBeInTheDocument()
    await userEvent.click(within(parent).getByRole("button", { name: "编辑 Inline 规则 1" }))

    const child = await screen.findByRole("dialog", { name: "编辑 Inline 规则 1" })
    fireEvent.change(within(child).getByLabelText("查询类型"), { target: { value: "1\nAAAA" } })
    await userEvent.click(within(child).getByRole("tab", { name: "域名与地址" }))
    fireEvent.change(within(child).getByLabelText("域名后缀"), { target: { value: "internal.example\nexample.net" } })
    await userEvent.click(within(child).getByRole("button", { name: "保存" }))
    await userEvent.click(within(parent).getByRole("button", { name: "保存" }))

    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({
      type: "inline", tag: "private",
      rules: [{ query_type: [1, "AAAA"], domain_suffix: ["internal.example", "example.net"] }],
    }))
  })

  it("copies, reorders, and confirms deletion without aliasing rules", async () => {
    renderDialog(<InlineHarness initial={{
      type: "inline", tag: "inline", rules: [{ domain: ["a.example"] }, { port: [443] }],
    }} />)

    await userEvent.click(screen.getByRole("button", { name: "复制 Inline 规则 1" }))
    expect(screen.getAllByRole("button", { name: /编辑 Inline 规则/ })).toHaveLength(3)
    await userEvent.click(screen.getByRole("button", { name: "下移 Inline 规则 1" }))
    await userEvent.click(screen.getByRole("button", { name: "删除 Inline 规则 2" }))
    const confirm = await screen.findByRole("alertdialog")
    await userEvent.click(within(confirm).getByRole("button", { name: "确认删除" }))

    expect(JSON.parse(screen.getByLabelText("inline state").textContent ?? "{}").rules).toEqual([
      { domain: ["a.example"] }, { port: [443] },
    ])
  })

  it("requires a meaningful match and supports logical child rules", async () => {
    const onSave = vi.fn()
    renderDialog(<RouteHeadlessRuleDialog open title="新增 Inline 规则" item={{}}
      onOpenChange={vi.fn()} onSave={onSave} />)

    const dialog = screen.getByRole("dialog", { name: "新增 Inline 规则" })
    expect(within(dialog).getByRole("button", { name: "保存" })).toBeDisabled()
    await choose(dialog, "规则类型", "logical")
    expect(within(dialog).queryByRole("tab", { name: "域名与地址" })).not.toBeInTheDocument()
    await choose(dialog, "逻辑模式", "or")
    fireEvent.change(within(dialog).getByLabelText("子规则 JSON"), {
      target: { value: '[{"domain":["example.com"]}]' },
    })
    await userEvent.click(within(dialog).getByRole("button", { name: "保存" }))

    expect(onSave).toHaveBeenCalledWith({
      type: "logical", mode: "or", rules: [{ domain: ["example.com"] }],
    })
  })
})
