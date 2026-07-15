import { fireEvent, screen } from "@testing-library/react"
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
    fireEvent.change(screen.getByLabelText("服务器地址"), { target: { value: "new.example.com" } })
    fireEvent.change(screen.getByLabelText("服务器端口"), { target: { value: "8443" } })
    await user.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ server: "new.example.com", server_port: 8443 }))
  })

  it("disables saving when the JSON root is not an object", async () => {
    const user = userEvent.setup()
    renderApp(<ProxyEditorDialog title="编辑出站" kind="outbounds" item={{ tag: "node" }} onClose={vi.fn()} onSave={vi.fn()} />)
    await user.click(screen.getByRole("tab", { name: "高级 JSON" }))
    const editor = screen.getByRole("textbox", { name: "编辑出站 JSON" })
    await user.click(editor)
    await user.keyboard("{Control>}a{/Control}[BracketLeft][BracketRight]")
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
  })

  it("uses outbound address fields for a new outbound", async () => {
    const onSave = vi.fn()
    const user = userEvent.setup()
    renderApp(<ProxyEditorDialog title="新增出站" kind="outbounds" item={{}} onClose={vi.fn()} onSave={onSave} />)
    await user.click(screen.getByRole("combobox", { name: "类型" }))
    await user.click(await screen.findByRole("option", { name: "vless" }))
    fireEvent.change(screen.getByLabelText("服务器地址"), { target: { value: "new.example.com" } })
    fireEvent.change(screen.getByLabelText("服务器端口"), { target: { value: "443" } })
    await user.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ server: "new.example.com", server_port: 443 }))
  })
})
