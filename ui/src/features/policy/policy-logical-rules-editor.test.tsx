import { useState } from "react"
import { screen, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it } from "vitest"

import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import {
  PolicyLogicalRulesEditor,
  type PolicyLogicalRuleEditorRenderProps,
} from "@/features/policy/policy-logical-rules-editor"
import type { JsonObject } from "@/features/policy/policy-form-model"
import { renderApp } from "@/test/render"

function TestDialog({ item, title, onOpenChange, onSave }: PolicyLogicalRuleEditorRenderProps) {
  return <Dialog open onOpenChange={onOpenChange}><DialogContent>
    <DialogHeader><DialogTitle>{title}</DialogTitle><DialogDescription>test child editor</DialogDescription></DialogHeader>
    <Button onClick={() => onSave({ ...item, action: "reject" })}>save child</Button>
    <Button variant="outline" onClick={() => onOpenChange(false)}>close child</Button>
  </DialogContent></Dialog>
}

function Harness({ initial = [] }: { initial?: JsonObject[] }) {
  const [rules, setRules] = useState(initial)
  return <>
    <PolicyLogicalRulesEditor section="route" rules={rules} onChange={setRules}
      summarize={(rule) => ({
        matches: Array.isArray(rule.matches)
          ? rule.matches.filter((value): value is string => typeof value === "string")
          : typeof rule.domain === "string" ? [rule.domain] : [],
        action: String(rule.action ?? "route"),
      })}
      renderEditor={(props) => <TestDialog {...props} />} />
    <output aria-label="logical rules state">{JSON.stringify(rules)}</output>
    <output aria-label="logical rules identity">{String(rules.length > 1 && rules[0] === rules[1])}</output>
  </>
}

describe("PolicyLogicalRulesEditor", () => {
  it("adds a child rule from the empty state", async () => {
    const user = userEvent.setup()
    renderApp(<Harness />)

    expect(screen.getByText("暂无逻辑子规则")).toBeInTheDocument()
    await user.click(screen.getAllByRole("button", { name: "新增子规则" })[0])
    const dialog = within(screen.getByRole("dialog"))
    expect(dialog.getByRole("heading", { name: "新增子规则" })).toBeInTheDocument()
    await user.click(dialog.getByRole("button", { name: "save child" }))

    expect(screen.getByLabelText("logical rules state")).toHaveTextContent('[{"action":"reject"}]')
    expect(screen.getByText("reject", { selector: '[data-slot="badge"]' })).toBeInTheDocument()
  })

  it("edits, copies deeply, reorders, and deletes child rules", async () => {
    const user = userEvent.setup()
    renderApp(<Harness initial={[{ domain: "one.example", payload: { keep: true } }, { action: "reject" }]} />)

    await user.click(screen.getByRole("button", { name: "编辑子规则 1" }))
    await user.click(within(screen.getByRole("dialog")).getByRole("button", { name: "save child" }))
    await user.click(screen.getByRole("button", { name: "复制子规则 1" }))
    expect(screen.getByLabelText("logical rules identity")).toHaveTextContent("false")

    await user.click(screen.getByRole("button", { name: "下移子规则 1" }))
    expect(screen.getByLabelText("logical rules state")).toHaveTextContent(
      '[{"domain":"one.example","payload":{"keep":true},"action":"reject"},{"domain":"one.example","payload":{"keep":true},"action":"reject"},{"action":"reject"}]',
    )
    await user.click(screen.getByRole("button", { name: "上移子规则 2" }))

    await user.click(screen.getByRole("button", { name: "删除子规则 2" }))
    await user.click(screen.getByRole("button", { name: "确认删除" }))
    expect(screen.getByLabelText("logical rules state")).toHaveTextContent(
      '[{"domain":"one.example","payload":{"keep":true},"action":"reject"},{"action":"reject"}]',
    )
  })

  it("summarizes nested rules, caps match badges, and closes without saving", async () => {
    const user = userEvent.setup()
    renderApp(<Harness initial={[{
      type: "logical", action: "reject", rules: [{ action: "reject" }],
      matches: ["one", "two", "three", "four", "five"],
    }]} />)

    expect(screen.getByText("包含 1 条嵌套子规则")).toBeInTheDocument()
    expect(screen.getByText("+1", { selector: '[data-slot="badge"]' })).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "新增子规则" }))
    await user.click(within(screen.getByRole("dialog")).getByRole("button", { name: "close child" }))
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
    expect(screen.getByLabelText("logical rules state")).toHaveTextContent('"type":"logical"')
  })
})
