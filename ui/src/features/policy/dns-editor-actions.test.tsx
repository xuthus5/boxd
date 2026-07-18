import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { useState } from "react"
import { fireEvent, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"

import { DNSServerCard } from "@/features/policy/dns-server-card"
import { DNSRuleCard } from "@/features/policy/dns-rule-card"
import { DNSVisualEditor } from "@/features/policy/dns-visual-editor"
import type { JsonObject } from "@/features/policy/policy-form-model"
import type { PolicyVisualEditorProps } from "@/features/policy/policy-page"
import { renderApp } from "@/test/render"

function EditorHarness({ initial }: { initial: JsonObject }) {
  const [object, setObject] = useState(initial)
  const props: PolicyVisualEditorProps = {
    object, revision: 0, onChange: setObject, onFieldValidityChange: vi.fn(),
  }
  return <><DNSVisualEditor {...props} /><output aria-label="dns action state">{JSON.stringify(object)}</output></>
}

function state(): JsonObject {
  return JSON.parse(screen.getByLabelText("dns action state").textContent ?? "{}") as JsonObject
}

async function choose(label: string, option: string) {
  await userEvent.click(screen.getByRole("combobox", { name: label }))
  await userEvent.click(await screen.findByRole("option", { name: option }))
}

function renderEditor(ui: React.ReactElement) {
  return renderApp(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>{ui}</QueryClientProvider>)
}

describe("DNS visual editor replacement workflows", () => {
  it("replaces existing servers and rules and closes a cancelled dialog", async () => {
    const user = userEvent.setup()
    renderEditor(<EditorHarness initial={{
      servers: [{ tag: "local", address: "local" }],
      rules: [{ action: "reject", domain: ["old.example"] }],
    }} />)

    await user.click(screen.getByRole("button", { name: "编辑 DNS 服务器 local" }))
    fireEvent.change(screen.getByLabelText("旧式地址"), { target: { value: "tcp://127.0.0.1" } })
    await user.click(screen.getByRole("button", { name: "保存" }))
    expect((state().servers as JsonObject[])[0]).toMatchObject({ address: "tcp://127.0.0.1" })

    await user.click(screen.getByRole("button", { name: "编辑 DNS 规则 1" }))
    await user.click(screen.getByRole("tab", { name: "执行动作" }))
    await choose("执行动作", "predefined")
    fireEvent.change(screen.getByLabelText("响应码"), { target: { value: "REFUSED" } })
    await user.click(screen.getByRole("button", { name: "保存" }))
    expect((state().rules as JsonObject[])[0]).toMatchObject({ action: "predefined", rcode: "REFUSED" })

    await user.click(screen.getByRole("button", { name: "编辑 DNS 服务器 local" }))
    await user.click(screen.getByRole("button", { name: "取消" }))
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
  }, 15_000)

  it("appends a route rule from Empty using an existing server tag", async () => {
    const user = userEvent.setup()
    renderEditor(<EditorHarness initial={{ servers: [{ tag: "dns", address: "local" }] }} />)
    await user.click(screen.getAllByRole("button", { name: "新增 DNS 规则" })[0])
    await user.click(screen.getByRole("tab", { name: "执行动作" }))
    await choose("目标 DNS 服务器", "dns")
    await user.click(screen.getByRole("button", { name: "保存" }))
    expect(state().rules).toEqual([{ action: "route", server: "dns" }])
  })

  it("synchronizes a server card after Advanced JSON replacement", async () => {
    const user = userEvent.setup()
    renderEditor(<EditorHarness initial={{ servers: [{ tag: "old", address: "local" }] }} />)
    await user.click(screen.getByRole("button", { name: "编辑 DNS 服务器 old" }))
    await user.click(screen.getByRole("tab", { name: "高级 JSON" }))
    const editor = screen.getByRole("textbox", { name: "编辑 DNS 服务器 JSON" })
    await user.click(editor)
    await user.keyboard("{Control>}a{/Control}")
    await user.paste('{"type":"future","tag":"new","payload":{"keep":true}}')
    await user.click(screen.getByRole("button", { name: "保存" }))
    expect(screen.getByRole("button", { name: "编辑 DNS 服务器 new" })).toBeInTheDocument()
    expect((state().servers as JsonObject[])[0]).toEqual({ type: "future", tag: "new", payload: { keep: true } })
  })
})

describe("DNS mobile secondary actions", () => {
  it("runs server copy and delete from the DropdownMenu", async () => {
    const onCopy = vi.fn()
    const onDelete = vi.fn()
    renderApp(<DNSServerCard item={{ tag: "dns", address: "local" }} onEdit={vi.fn()} onCopy={onCopy} onDelete={onDelete} />)
    await userEvent.click(screen.getByRole("button", { name: "更多 DNS 服务器 dns" }))
    await userEvent.click(await screen.findByRole("menuitem", { name: "复制" }))
    expect(onCopy).toHaveBeenCalledOnce()
    await userEvent.click(screen.getByRole("button", { name: "更多 DNS 服务器 dns" }))
    await userEvent.click(await screen.findByRole("menuitem", { name: "删除" }))
    await userEvent.click(screen.getByRole("button", { name: "确认删除" }))
    expect(onDelete).toHaveBeenCalledOnce()
  })

  it("runs rule copy, movement, and delete from the DropdownMenu", async () => {
    const handlers = { onEdit: vi.fn(), onCopy: vi.fn(), onMoveUp: vi.fn(), onMoveDown: vi.fn(), onDelete: vi.fn() }
    renderApp(<DNSRuleCard index={1} item={{ action: "reject" }} first={false} last={false} {...handlers} />)
    for (const [name, callback] of [["复制", handlers.onCopy], ["上移", handlers.onMoveUp], ["下移", handlers.onMoveDown]] as const) {
      await userEvent.click(screen.getByRole("button", { name: "更多 DNS 规则 2" }))
      await userEvent.click(await screen.findByRole("menuitem", { name }))
      expect(callback).toHaveBeenCalledOnce()
    }
    await userEvent.click(screen.getByRole("button", { name: "更多 DNS 规则 2" }))
    await userEvent.click(await screen.findByRole("menuitem", { name: "删除" }))
    await userEvent.click(screen.getByRole("button", { name: "确认删除" }))
    expect(handlers.onDelete).toHaveBeenCalledOnce()
  })
})
