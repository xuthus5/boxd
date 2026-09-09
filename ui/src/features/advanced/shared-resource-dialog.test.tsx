import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { fireEvent, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { SharedResourceDialog } from "@/features/advanced/shared-resource-dialog"
import { installMockAPI } from "@/test/mock-api"
import { renderApp } from "@/test/render"

beforeEach(() => { installMockAPI() })

function setup() {
  const onSave = vi.fn()
  const onClose = vi.fn()
  renderApp(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
    <SharedResourceDialog section="certificate_providers" item={{ type: "acme", tag: "managed" }} onSave={onSave} onClose={onClose} />
  </QueryClientProvider>)
  return { user: userEvent.setup(), onSave, onClose }
}

async function replaceJSON(raw: string) {
  const user = userEvent.setup()
  const editor = screen.getByRole("textbox", { name: "共享资源 JSON" })
  await user.click(editor)
  await user.keyboard("{Control>}a{/Control}")
  await user.paste(raw)
}

describe("shared resource JSON editing", () => {
  it("replaces a provider through Advanced JSON and preserves its complete options", async () => {
    const { user, onSave } = setup()
    await user.click(screen.getByRole("tab", { name: "高级 JSON" }))
    const object = { type: "cloudflare-origin-ca", tag: "origin", domain: ["example.org"], origin_ca_key: "test-key", request_type: "origin-ecc" }
    await replaceJSON(JSON.stringify(object))
    await user.click(screen.getByRole("tab", { name: "可视化配置" }))
    expect(screen.getByLabelText("Origin CA 密钥")).toHaveValue("test-key")
    await user.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenCalledWith(object)
  })

  it("blocks malformed root JSON across tabs and recovers after replacement", async () => {
    const { user, onSave } = setup()
    await user.click(screen.getByRole("tab", { name: "高级 JSON" }))
    await replaceJSON("{")
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    await user.click(screen.getByRole("tab", { name: "可视化配置" }))
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    await user.click(screen.getByRole("tab", { name: "高级 JSON" }))
    await replaceJSON('{"type":"tailscale","tag":"managed","endpoint":"vpn"}')
    await user.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenCalledWith({ type: "tailscale", tag: "managed", endpoint: "vpn" })
  })

  it("clears structured field errors when valid root JSON replaces the draft", async () => {
    const { user, onClose } = setup()
    fireEvent.change(screen.getByLabelText("DNS-01 验证配置"), { target: { value: "{" } })
    await user.click(screen.getByRole("tab", { name: "高级 JSON" }))
    await replaceJSON('{"type":"tailscale","tag":"managed","endpoint":"vpn"}')
    expect(screen.getByRole("button", { name: "保存" })).toBeEnabled()
    await user.keyboard("{Escape}")
    expect(onClose).toHaveBeenCalledOnce()
  })
})
