import { cleanup, fireEvent, render, screen, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"

import { WireGuardPeerEditor } from "@/features/advanced/wireguard-peer-editor"
import { i18n } from "@/i18n"

function renderEditor(value: unknown, onChange = vi.fn()) {
  return {
    onChange,
    view: render(
      <I18nextProvider i18n={i18n}>
        <WireGuardPeerEditor value={value as never} onChange={onChange} />
      </I18nextProvider>,
    ),
  }
}

describe("WireGuardPeerEditor", () => {
  it("adds a valid peer from the visual dialog", async () => {
    const user = userEvent.setup()
    const { onChange } = renderEditor([])
    await user.click(screen.getAllByRole("button", { name: "新增 Peer" })[0])
    const dialog = await screen.findByRole("dialog")
    fireEvent.change(within(dialog).getByLabelText("Peer 公钥"), { target: { value: "public-key" } })
    fireEvent.change(within(dialog).getByLabelText("允许的 IP"), { target: { value: "0.0.0.0/0" } })
    expect(within(dialog).getByRole("button", { name: "保存" })).toBeEnabled()
    await user.click(within(dialog).getByRole("button", { name: "保存" }))
    expect(onChange).toHaveBeenCalledWith([{ public_key: "public-key", allowed_ips: ["0.0.0.0/0"] }])
  })

  it("blocks incomplete peers and reports malformed lists", async () => {
    const user = userEvent.setup()
    const onValidityChange = vi.fn()
    render(
      <I18nextProvider i18n={i18n}>
        <WireGuardPeerEditor value={["bad"] as never} onChange={vi.fn()} onValidityChange={onValidityChange} />
      </I18nextProvider>,
    )
    expect(screen.getByText("每个 Peer 必须是对象，请在高级 JSON 中修复 Peers。" )).toBeInTheDocument()
    expect(onValidityChange).toHaveBeenCalledWith(false)

    cleanup()
    expect(onValidityChange).toHaveBeenLastCalledWith(true)

    const { onChange } = renderEditor([])
    await user.click(screen.getAllByRole("button", { name: "新增 Peer" })[1])
    const dialog = await screen.findByRole("dialog")
    fireEvent.change(within(dialog).getByLabelText("Peer 公钥"), { target: { value: "public-key" } })
    expect(within(dialog).getByRole("button", { name: "保存" })).toBeDisabled()
    expect(onChange).not.toHaveBeenCalled()
  })

  it("edits peers and recovers from invalid transport fields", async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    renderEditor([
      { public_key: "first-key", allowed_ips: ["10.0.0.0/8"], custom: { keep: true } },
      { public_key: "second-key", allowed_ips: ["fd00::/8"] },
    ], onChange)

    await user.click(screen.getByRole("button", { name: "编辑 Peer 1" }))
    const dialog = await screen.findByRole("dialog", { name: "编辑 WireGuard Peer" })
    fireEvent.change(within(dialog).getByLabelText("Peer 公钥"), { target: { value: "updated-key" } })
    const port = within(dialog).getByLabelText("Peer 端口")
    fireEvent.change(port, { target: { value: "65536" } })
    expect(within(dialog).getByRole("button", { name: "保存" })).toBeDisabled()
    expect(port).toHaveAttribute("aria-invalid", "true")

    fireEvent.change(port, { target: { value: "51820" } })
    expect(port).toHaveAttribute("aria-invalid", "false")
    expect(within(dialog).getByRole("button", { name: "保存" })).toBeEnabled()
    await user.click(within(dialog).getByRole("button", { name: "保存" }))
    expect(onChange).toHaveBeenCalledWith([
      { public_key: "updated-key", allowed_ips: ["10.0.0.0/8"], port: 51820, custom: { keep: true } },
      { public_key: "second-key", allowed_ips: ["fd00::/8"] },
    ])
  })

  it("closes peer dialogs through cancel and the dialog close action", async () => {
    const user = userEvent.setup()
    renderEditor([])
    await user.click(screen.getAllByRole("button", { name: "新增 Peer" })[0])
    await user.click(within(await screen.findByRole("dialog")).getByRole("button", { name: "取消" }))
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument()

    await user.click(screen.getAllByRole("button", { name: "新增 Peer" })[0])
    expect(await screen.findByRole("dialog")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "Close" }))
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
  })

  it("copies and deletes peers", async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    renderEditor([{ public_key: "key", allowed_ips: ["10.0.0.0/8"] }], onChange)
    await user.click(screen.getByRole("button", { name: "复制 Peer 1" }))
    expect(onChange).toHaveBeenCalledWith([
      { public_key: "key", allowed_ips: ["10.0.0.0/8"] },
      { public_key: "key", allowed_ips: ["10.0.0.0/8"] },
    ])
    await user.click(screen.getByRole("button", { name: "删除 Peer 1" }))
    await user.click(screen.getByRole("button", { name: "确认删除" }))
    expect(onChange).toHaveBeenLastCalledWith([])
  })

  it("renders endpoint, keepalive, and missing-key summaries", () => {
    renderEditor([
      {
        address: "peer.example",
        port: 51820,
        public_key: "endpoint-key",
        allowed_ips: ["10.0.0.0/8"],
        persistent_keepalive_interval: 25,
      },
      { allowed_ips: ["fd00::/8"] },
    ])
    expect(screen.getByText("peer.example:51820")).toBeInTheDocument()
    expect(screen.getByText("Keepalive 25 秒")).toBeInTheDocument()
    expect(screen.getByText("缺少公钥")).toBeInTheDocument()
  })
})
