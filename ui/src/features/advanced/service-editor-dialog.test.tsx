import { fireEvent, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

vi.mock("@/features/config/config-hooks", () => ({
  useConfigQuery: () => ({
    data: {
      inbounds: [{ type: "shadowsocks", tag: "ss-in" }, { type: "tun", tag: "tun-in" }],
      outbounds: [{ type: "direct", tag: "direct" }, { type: "selector", tag: "proxy" }],
    },
  }),
}))

import { ServiceEditorDialog } from "@/features/advanced/service-editor-dialog"
import type { JsonObject } from "@/features/policy/policy-form-model"
import { renderApp } from "@/test/render"

afterEach(() => {
  vi.clearAllMocks()
})

function renderDialog(item: JsonObject, onSave = vi.fn()) {
  const onOpenChange = vi.fn()
  const view = renderApp(
    <ServiceEditorDialog
      open
      item={item}
      title="编辑内核服务"
      onOpenChange={onOpenChange}
      onSave={onSave}
    />,
  )
  return { onOpenChange, onSave, user: userEvent.setup(), view }
}

async function selectType(user: ReturnType<typeof userEvent.setup>, type: string) {
  await user.click(screen.getByRole("combobox", { name: "服务类型" }))
  await user.click(await screen.findByRole("option", { name: type }))
}

describe("ServiceEditorDialog", () => {
  it("shows all service types and their specific fields", async () => {
    const { user } = renderDialog({ type: "resolved" })
    const dialog = screen.getByRole("dialog")
    expect(within(dialog).getByLabelText("监听地址")).toHaveValue("127.0.0.53")
    expect(within(dialog).getByLabelText("监听端口")).toHaveValue(53)
    expect(within(dialog).queryByRole("switch", { name: "启用 TLS" })).not.toBeInTheDocument()

    await user.click(within(dialog).getByRole("combobox", { name: "服务类型" }))
    for (const type of ["ccm", "derp", "ocm", "resolved", "ssm-api"]) {
      expect(await screen.findByRole("option", { name: type })).toBeInTheDocument()
    }
    await user.click(screen.getByRole("option", { name: "ccm" }))
    expect(within(dialog).getByLabelText("OAuth 凭据路径")).toBeInTheDocument()
    expect(within(dialog).getByLabelText("授权用户 JSON")).toBeInTheDocument()
    expect(within(dialog).getByRole("switch", { name: "启用 TLS" })).toBeInTheDocument()

    await selectType(user, "ocm")
    expect(within(dialog).getByLabelText("上游请求头 JSON")).toBeInTheDocument()

    await selectType(user, "derp")
    expect(within(dialog).getByLabelText("DERP 配置路径")).toBeInTheDocument()
    expect(within(dialog).getByLabelText("STUN 配置 JSON")).toBeInTheDocument()

    await selectType(user, "ssm-api")
    expect(within(dialog).getByLabelText("Shadowsocks 映射 JSON")).toBeInTheDocument()
    expect(within(dialog).getByLabelText("SSM 状态缓存路径")).toBeInTheDocument()
  })

  it("enforces DERP and SSM API requirements before saving", async () => {
    const derpSave = vi.fn()
    const derp = renderDialog({ type: "derp", listen: "::", listen_port: 443 }, derpSave)
    let dialog = screen.getByRole("dialog")
    expect(within(dialog).getByRole("button", { name: "保存" })).toBeDisabled()
    fireEvent.change(within(dialog).getByLabelText("DERP 配置路径"), { target: { value: "derper.key" } })
    expect(within(dialog).getByRole("button", { name: "保存" })).toBeEnabled()
    await derp.user.click(within(dialog).getByRole("button", { name: "保存" }))
    expect(derpSave).toHaveBeenCalledWith(expect.objectContaining({ config_path: "derper.key" }))
    derp.view.unmount()

    const ssmSave = vi.fn()
    const ssm = renderDialog({ type: "ssm-api", listen: "127.0.0.1", listen_port: 8080 }, ssmSave)
    dialog = screen.getByRole("dialog")
    expect(within(dialog).getByRole("button", { name: "保存" })).toBeDisabled()
    fireEvent.change(within(dialog).getByLabelText("Shadowsocks 映射 JSON"), {
      target: { value: "{\"/\":\"ss-in\"}" },
    })
    await waitFor(() => expect(within(dialog).getByRole("button", { name: "保存" })).toBeEnabled())
    await ssm.user.click(within(dialog).getByRole("button", { name: "保存" }))
    expect(ssmSave).toHaveBeenCalledWith(expect.objectContaining({ servers: { "/": "ss-in" } }))
  })

  it("blocks malformed advanced JSON and saves a valid replacement", async () => {
    const onSave = vi.fn()
    const { user } = renderDialog({ type: "resolved" }, onSave)
    await user.click(screen.getByRole("tab", { name: "高级 JSON" }))
    const editor = await screen.findByLabelText("内核服务 JSON")
    await user.click(editor)
    await user.keyboard("{Control>}a{/Control}not-json")
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()

    await user.click(editor)
    await user.keyboard("{Control>}a{/Control}")
    await user.paste('{"type":"derp","listen":"::","listen_port":443,"config_path":"derper.key","future_option":true}')
    await waitFor(() => expect(screen.getByRole("button", { name: "保存" })).toBeEnabled())
    await user.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenCalledWith({
      type: "derp",
      listen: "::",
      listen_port: 443,
      config_path: "derper.key",
      future_option: true,
    })
  })
})
