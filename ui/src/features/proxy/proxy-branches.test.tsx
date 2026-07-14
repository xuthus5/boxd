import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"

import { ProxyEditorDialog } from "@/features/proxy/proxy-editor-dialog"
import { renderApp } from "@/test/render"

describe("ProxyEditorDialog states", () => {
  it("edits outbound address fields and closes", async () => {
    const onClose = vi.fn()
    const onSave = vi.fn()
    const user = userEvent.setup()
    renderApp(<ProxyEditorDialog title="编辑出站" kind="outbounds" item={{ tag: "node", type: "vless", server: "old", server_port: 443 }} onClose={onClose} onSave={onSave} />)
    await user.clear(screen.getByLabelText("地址"))
    await user.type(screen.getByLabelText("地址"), "new.example.com")
    await user.clear(screen.getByLabelText("端口"))
    await user.type(screen.getByLabelText("端口"), "8443")
    await user.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ server: "new.example.com", server_port: 8443 }))
  })

  it("disables saving when the JSON root is not an object", async () => {
    const user = userEvent.setup()
    renderApp(<ProxyEditorDialog title="编辑出站" kind="outbounds" item={{ tag: "node" }} onClose={vi.fn()} onSave={vi.fn()} />)
    const editor = screen.getByRole("textbox", { name: "编辑出站 JSON" })
    await user.click(editor)
    await user.keyboard("{Control>}a{/Control}[BracketLeft][BracketRight]")
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
  })

  it("uses outbound address fields for a new outbound", async () => {
    const onSave = vi.fn()
    const user = userEvent.setup()
    renderApp(<ProxyEditorDialog title="新增出站" kind="outbounds" item={{}} onClose={vi.fn()} onSave={onSave} />)
    await user.type(screen.getByLabelText("地址"), "new.example.com")
    await user.type(screen.getByLabelText("端口"), "443")
    await user.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ server: "new.example.com", server_port: 443 }))
  })
})
