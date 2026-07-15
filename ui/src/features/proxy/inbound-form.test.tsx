import { useState } from "react"
import { cleanup, fireEvent, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"

import { InboundFormFields } from "@/features/proxy/inbound-form-fields"
import {
  changeInboundType, changeTransportType, getPath, protocolFields, setPath, transportTypeFields, type FieldSpec, type JsonObject,
} from "@/features/proxy/inbound-form-model"
import { ProxyFormFields } from "@/features/proxy/proxy-form-fields"
import { renderApp } from "@/test/render"

describe("inbound form model", () => {
  it("reads, writes, and prunes nested values", () => {
    expect(getPath({ tls: { enabled: true } }, "tls.enabled")).toBe(true)
    expect(getPath({ tls: [] }, "tls.enabled")).toBeUndefined()
    expect(setPath({}, "tag", "in")).toEqual({ tag: "in" })
    expect(setPath({ tag: "in" }, "tag", undefined)).toEqual({})
    expect(setPath({}, "tls.reality.enabled", true)).toEqual({ tls: { reality: { enabled: true } } })
    expect(setPath({ tls: "invalid" }, "tls.enabled", true)).toEqual({ tls: { enabled: true } })
    expect(setPath({ tls: { enabled: true, server_name: "host" } }, "tls.enabled", undefined)).toEqual({ tls: { server_name: "host" } })
    expect(setPath({ tls: { enabled: true } }, "tls.enabled", undefined)).toEqual({})
  })

  it("cleans managed fields and handles unknown metadata", () => {
    expect(changeInboundType({ type: "mixed", listen: "::", tls: {}, custom: "keep" }, "tun")).toEqual({ type: "tun", custom: "keep" })
    expect(protocolFields("unknown")).toEqual([])
    expect(transportTypeFields("unknown")).toHaveLength(1)
  })

  it("preserves shared and unknown fields across type changes", () => {
    const object = { type: "vless", listen: "::", listen_port: 443, users: [{ uuid: "old" }], tls: { enabled: true, custom: "keep" }, transport: { type: "ws", path: "/ws", custom: "keep" }, multiplex: { enabled: true, custom: "keep" }, custom: "keep" }
    expect(changeInboundType(object, "trojan")).toEqual({
      type: "trojan", listen: "::", listen_port: 443, tls: { enabled: true, custom: "keep" },
      transport: { type: "ws", path: "/ws", custom: "keep" }, multiplex: { enabled: true, custom: "keep" }, custom: "keep",
    })
    expect(changeInboundType(object, "vless")).toBe(object)
    expect(changeInboundType({ type: "vless", tls: {}, transport: {}, multiplex: {} }, "direct")).toEqual({ type: "direct" })
    expect(changeInboundType({ type: "tun", interface_name: "tun0", udp_timeout: "5m" }, "mixed")).toEqual({ type: "mixed", udp_timeout: "5m" })
  })

  it("cleans incompatible transport fields and keeps shared headers", () => {
    const object = { transport: { type: "http", host: ["a"], path: "/old", method: "GET", headers: { X: "1" }, idle_timeout: "1m", custom: "keep" } }
    expect(changeTransportType(object, "httpupgrade")).toEqual({ transport: { type: "httpupgrade", path: "/old", headers: { X: "1" }, custom: "keep" } })
    expect(changeTransportType({ transport: { type: "ws", path: "/ws", custom: "keep" } }, "")).toEqual({ transport: { custom: "keep" } })
    expect(changeTransportType({}, "quic")).toEqual({ transport: { type: "quic" } })
  })
})

function renderFields(fields: FieldSpec[], object: JsonObject, type = "mixed") {
  const onChange = vi.fn()
  renderApp(<InboundFormFields fields={fields} object={object} type={type} onChange={onChange} />)
  return onChange
}

function JSONFieldHarness() {
  const [object, setObject] = useState<JsonObject>({ transport: { headers: { X: "old" } } })
  return <>
    <button onClick={() => setObject({ transport: { headers: { X: "new" } } })}>External update</button>
    <InboundFormFields fields={[{ path: "transport.headers", label: "transportHeaders", kind: "json-object" }]} object={object} type="vless" onChange={setObject} />
  </>
}

function JSONFieldRevisionHarness() {
  const [object, setObject] = useState<JsonObject>({ transport: { headers: { X: "old" } } })
  const [revision, setRevision] = useState(0)
  const [valid, setValid] = useState(true)
  return <>
    <button onClick={() => { setObject((current) => ({ ...current, tag: "changed" })); setRevision((current) => current + 1) }}>JSON update</button>
    <span>{valid ? "valid" : "invalid"}</span>
    <ProxyFormFields fields={[{ path: "transport.headers", label: "transportHeaders", kind: "json-object" }]} object={object} namespace="proxy.inbound" onChange={setObject} onFieldValidityChange={(_path, next) => setValid(next)} revision={revision} />
  </>
}

describe("inbound form field conversions", () => {
  it("updates lists, numeric lists, numbers, text, and booleans", async () => {
    const user = userEvent.setup()
    const listChange = renderFields([{ path: "address", label: "tunAddress", kind: "list" }], { address: ["10.0.0.1/30"] }, "tun")
    fireEvent.change(screen.getByLabelText("接口地址"), { target: { value: "10.0.0.1/30, fd00::1/126" } })
    expect(listChange).toHaveBeenLastCalledWith({ address: ["10.0.0.1/30", "fd00::1/126"] })
    fireEvent.change(screen.getByLabelText("接口地址"), { target: { value: "" } })
    expect(listChange).toHaveBeenLastCalledWith({})
    cleanup()

    const numberListChange = renderFields([{ path: "include_uid", label: "includeUID", kind: "number-list" }], {})
    fireEvent.change(screen.getByLabelText("包含 UID"), { target: { value: "1000,1001" } })
    expect(numberListChange).toHaveBeenLastCalledWith({ include_uid: [1000, 1001] })
    fireEvent.change(screen.getByLabelText("包含 UID"), { target: { value: "invalid" } })
    expect(numberListChange).toHaveBeenCalledTimes(1)
    cleanup()

    const emptyNumberListChange = renderFields([{ path: "include_uid", label: "includeUID", kind: "number-list" }], { include_uid: [1000] })
    fireEvent.change(screen.getByLabelText("包含 UID"), { target: { value: "" } })
    expect(emptyNumberListChange).toHaveBeenLastCalledWith({})
    cleanup()

    const numberChange = renderFields([{ path: "mtu", label: "mtu", kind: "number" }], { mtu: 9000 })
    fireEvent.change(screen.getByLabelText("MTU"), { target: { value: "" } })
    expect(numberChange).toHaveBeenLastCalledWith({})
    cleanup()

    const textChange = renderFields([{ path: "listen", label: "listenAddress" }], {})
    fireEvent.change(screen.getByLabelText("监听地址"), { target: { value: "::" } })
    expect(textChange).toHaveBeenLastCalledWith({ listen: "::" })
    cleanup()

    const booleanChange = renderFields([{ path: "auto_route", label: "autoRoute", kind: "boolean" }], { auto_route: true })
    await user.click(screen.getByRole("switch", { name: "自动路由" }))
    expect(booleanChange).toHaveBeenLastCalledWith({})
  })

  it("supports explicit UDP fragmentation and user variants", async () => {
    const user = userEvent.setup()
    const selectChange = renderFields([{ path: "udp_fragment", label: "udpFragment", kind: "select", options: ["true", "false"] }], { udp_fragment: false })
    await user.click(screen.getByRole("combobox", { name: "UDP 分片" }))
    await user.click(await screen.findByRole("option", { name: "启用" }))
    expect(selectChange).toHaveBeenLastCalledWith({ udp_fragment: true })
    await user.click(screen.getByRole("combobox", { name: "UDP 分片" }))
    await user.click(await screen.findByRole("option", { name: "未设置" }))
    expect(selectChange).toHaveBeenLastCalledWith({})
    cleanup()

    const usersChange = renderFields([{ path: "users", label: "users", kind: "users" }], { users: [{ name: "old", uuid: "id", custom: "keep" }] }, "vmess")
    expect(screen.getByLabelText("认证用户")).toHaveValue(JSON.stringify([{ name: "old", uuid: "id", custom: "keep" }], null, 2))
    fireEvent.change(screen.getByLabelText("认证用户"), { target: { value: '[{"name":"new","uuid":"new-id","custom":"keep"}]' } })
    expect(usersChange).toHaveBeenLastCalledWith({ users: [{ name: "new", uuid: "new-id", custom: "keep" }] })
    fireEvent.change(screen.getByLabelText("认证用户"), { target: { value: "invalid" } })
    expect(screen.getByText("请输入有效的 JSON 结构。")).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText("认证用户"), { target: { value: "" } })
    expect(usersChange).toHaveBeenLastCalledWith({})
  })

  it("validates JSON object fields", () => {
    const onChange = vi.fn()
    const onValidity = vi.fn()
    const view = renderApp(<InboundFormFields fields={[{ path: "transport.headers", label: "transportHeaders", kind: "json-object" }]} object={{}} type="vless" onChange={onChange} onFieldValidityChange={onValidity} />)
    const input = screen.getByLabelText("传输 Headers")
    fireEvent.change(input, { target: { value: "{}" } })
    expect(onChange).toHaveBeenLastCalledWith({ transport: { headers: {} } })
    fireEvent.change(input, { target: { value: "[]" } })
    expect(onValidity).toHaveBeenLastCalledWith("transport.headers", false)
    fireEvent.change(input, { target: { value: "" } })
    expect(onChange).toHaveBeenLastCalledWith({})
    view.unmount()
    expect(onValidity).toHaveBeenLastCalledWith("transport.headers", true)
  })

  it("synchronizes structured JSON fields after an external object update", async () => {
    const user = userEvent.setup()
    renderApp(<JSONFieldHarness />)
    expect(screen.getByLabelText("传输 Headers")).toHaveValue(JSON.stringify({ X: "old" }, null, 2))
    await user.click(screen.getByRole("button", { name: "External update" }))
    expect(screen.getByLabelText("传输 Headers")).toHaveValue(JSON.stringify({ X: "new" }, null, 2))
  })

  it("resets an invalid structured JSON draft when the form revision changes", async () => {
    const user = userEvent.setup()
    renderApp(<JSONFieldRevisionHarness />)
    fireEvent.change(screen.getByLabelText("传输 Headers"), { target: { value: "invalid" } })
    expect(screen.getByText("invalid")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "JSON update" }))
    expect(screen.getByLabelText("传输 Headers")).toHaveValue(JSON.stringify({ X: "old" }, null, 2))
    expect(screen.getByText("valid")).toBeInTheDocument()
  })
})
