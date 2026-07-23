import { fireEvent, screen } from "@testing-library/react"
import { useState } from "react"
import { describe, expect, it, vi } from "vitest"

import { OutboundFormFields } from "@/features/proxy/outbound-form-fields"
import type { JsonObject } from "@/features/proxy/proxy-form-model"
import { renderApp } from "@/test/render"

function NumberHarness({ onValidity }: { onValidity: (path: string, valid: boolean) => void }) {
  const [object, setObject] = useState<JsonObject>({ type: "vless", server_port: 443 })
  return (
    <OutboundFormFields
      fields={[{ path: "server_port", label: "serverPort", kind: "number", section: "basic" }]}
      object={object}
      type="vless"
      onChange={setObject}
      onFieldValidityChange={onValidity}
    />
  )
}

function ListHarness({ onValidity }: { onValidity: (path: string, valid: boolean) => void }) {
  const [object, setObject] = useState<JsonObject>({ type: "hysteria", server_ports: [443] })
  return (
    <OutboundFormFields
      fields={[{ path: "server_ports", label: "serverPorts", kind: "number-list", section: "protocol" }]}
      object={object}
      type="hysteria"
      onChange={setObject}
      onFieldValidityChange={onValidity}
    />
  )
}

describe("proxy number field validation", () => {
  it("marks invalid number input and blocks validity", () => {
    const onValidity = vi.fn()
    renderApp(<NumberHarness onValidity={onValidity} />)
    const input = screen.getByLabelText("服务器端口")
    fireEvent.change(input, { target: { value: "abc" } })
    expect(screen.getByText("请输入有效数字。")).toBeInTheDocument()
    expect(input).toHaveAttribute("aria-invalid", "true")
    expect(onValidity).toHaveBeenCalledWith("server_port", false)
    fireEvent.change(input, { target: { value: "8443" } })
    expect(screen.queryByText("请输入有效数字。")).not.toBeInTheDocument()
    expect(onValidity).toHaveBeenCalledWith("server_port", true)
  })

  it("marks invalid number lists", () => {
    const onValidity = vi.fn()
    renderApp(<ListHarness onValidity={onValidity} />)
    const input = screen.getByLabelText("服务器端口列表")
    fireEvent.change(input, { target: { value: "443,x" } })
    expect(screen.getByText("请输入以逗号或换行分隔的有效数字列表。")).toBeInTheDocument()
    expect(onValidity).toHaveBeenCalledWith("server_ports", false)
  })
})
