import { useState } from "react"
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { afterEach, describe, expect, it, vi } from "vitest"

import { InboundFormFields } from "@/features/proxy/inbound-form-fields"
import {
  applyInboundFieldChange, changeInboundType, changeTransportType, getPath, protocolFields, setPath, tlsFields, transportTypeFields, userSchema, type FieldSpec, type JsonObject,
} from "@/features/proxy/inbound-form-model"
import { configTags, dnsServerTags, groupFieldsBySection, isFieldVisible, pruneInvisibleFields } from "@/features/proxy/proxy-form-model"
import { ProxyFormFields } from "@/features/proxy/proxy-form-fields"
import { renderApp } from "@/test/render"

afterEach(() => vi.unstubAllGlobals())

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

function renderFields(fields: FieldSpec[], object: JsonObject, type = "mixed", revision = 0) {
  const onChange = vi.fn()
  renderApp(<InboundFormFields fields={fields} object={object} type={type} revision={revision} onChange={onChange} />)
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
function ListenRevisionHarness() {
  const [object, setObject] = useState<JsonObject>({ listen: "0.0.0.0" })
  const [revision, setRevision] = useState(0)
  return <>
    <button onClick={() => { setObject({ listen: "::" }); setRevision((current) => current + 1) }}>Listen update</button>
    <ProxyFormFields fields={[{ path: "listen", label: "listenAddress", kind: "listen-address" }]} object={object} namespace="proxy.inbound" onChange={setObject} revision={revision} />
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

    const listenChange = renderFields([{ path: "listen", label: "listenAddress", kind: "listen-address" }], {})
    await user.click(screen.getByRole("combobox", { name: "监听地址" }))
    await user.click(await screen.findByRole("option", { name: "0.0.0.0（IPv4 全接口）" }))
    expect(listenChange).toHaveBeenLastCalledWith({ listen: "0.0.0.0" })
    await user.click(screen.getByRole("combobox", { name: "监听地址" }))
    await user.click(await screen.findByRole("option", { name: "::（IPv6 全接口）" }))
    expect(listenChange).toHaveBeenLastCalledWith({ listen: "::" })
    await user.click(screen.getByRole("combobox", { name: "监听地址" }))
    await user.click(await screen.findByRole("option", { name: "手动输入" }))
    fireEvent.change(screen.getByLabelText("自定义监听地址"), { target: { value: "127.0.0.1" } })
    expect(listenChange).toHaveBeenLastCalledWith({ listen: "127.0.0.1" })
    await user.click(screen.getByRole("combobox", { name: "监听地址" }))
    await user.click(await screen.findByRole("option", { name: "未设置" }))
    expect(listenChange).toHaveBeenLastCalledWith({})
    cleanup()

    renderFields([{ path: "listen", label: "listenAddress", kind: "listen-address" }], { listen: "10.0.0.1" })
    expect(screen.getByLabelText("自定义监听地址")).toHaveValue("10.0.0.1")
    cleanup()

    const revisionView = renderApp(<InboundFormFields fields={[{ path: "listen", label: "listenAddress", kind: "listen-address" }]} object={{ listen: "0.0.0.0" }} type="mixed" revision={0} onChange={vi.fn()} />)
    expect(screen.queryByLabelText("自定义监听地址")).not.toBeInTheDocument()
    revisionView.rerender(<InboundFormFields fields={[{ path: "listen", label: "listenAddress", kind: "listen-address" }]} object={{ listen: "10.0.0.8" }} type="mixed" revision={1} onChange={vi.fn()} />)
    expect(screen.getByLabelText("自定义监听地址")).toHaveValue("10.0.0.8")
    cleanup()

    const booleanChange = renderFields([{ path: "auto_route", label: "autoRoute", kind: "boolean" }], { auto_route: true })
    await user.click(screen.getByRole("switch", { name: "自动路由" }))
    expect(booleanChange).toHaveBeenLastCalledWith({})
  })

  it("supports explicit UDP fragmentation and user variants", async () => {
    const user = userEvent.setup()
    const selectChange = renderFields([{ path: "udp_fragment", label: "udpFragment", kind: "boolean" }], { udp_fragment: true })
    await user.click(screen.getByRole("switch", { name: "UDP 分片" }))
    expect(selectChange).toHaveBeenLastCalledWith({})
    cleanup()

    const usersChange = renderFields([{ path: "users", label: "users", kind: "users" }], { users: [{ name: "old", uuid: "id", custom: "keep" }] }, "vmess")
    expect(screen.getByLabelText("认证用户 1 名称")).toHaveValue("old")
    expect(screen.getByLabelText("认证用户 1 UUID")).toHaveValue("id")
    fireEvent.change(screen.getByLabelText("认证用户 1 名称"), { target: { value: "new" } })
    expect(usersChange).toHaveBeenLastCalledWith({ users: [{ name: "new", uuid: "id", custom: "keep" }] })
    await user.click(screen.getByRole("button", { name: "删除用户" }))
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

  it("resets listen address mode when the form revision changes", async () => {
    const user = userEvent.setup()
    renderApp(<ListenRevisionHarness />)
    expect(screen.getByRole("combobox", { name: "监听地址" })).toHaveTextContent("0.0.0.0（IPv4 全接口）")
    await user.click(screen.getByRole("button", { name: "Listen update" }))
    expect(screen.getByRole("combobox", { name: "监听地址" })).toHaveTextContent("::（IPv6 全接口）")
  })
})


describe("inbound bind interface field", () => {
  it("loads network interfaces for bind interface selection and shows field help", async () => {
    const user = userEvent.setup()
    const fetchMock = vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path.includes("/api/network/interfaces")) {
        return Promise.resolve(new Response(JSON.stringify({
          status: "ok",
          data: {
            interfaces: [
              { name: "wlp3s0", ips: ["192.168.1.48"] },
              { name: "eno1", ips: [] },
              { name: "tun0" },
            ],
          },
          error: null,
          meta: null,
        }), { status: 200, headers: { "Content-Type": "application/json" } }))
      }
      return Promise.resolve(new Response(JSON.stringify({ status: "ok", data: {}, error: null, meta: null }), { status: 200 }))
    })
    vi.stubGlobal("fetch", fetchMock)
    const onChange = vi.fn()
    renderApp(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><InboundFormFields fields={[{ path: "bind_interface", label: "bindInterface", kind: "network-interface" }, { path: "routing_mark", label: "routingMark" }]} object={{}} type="mixed" onChange={onChange} /></QueryClientProvider>)
    expect((await screen.findAllByRole("button", { name: "字段说明" })).length).toBeGreaterThan(0)
    await screen.findByRole("combobox", { name: "绑定接口" })
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    await user.click(screen.getByRole("combobox", { name: "绑定接口" }))
    await user.click(await screen.findByRole("option", { name: "手动输入" }))
    expect(screen.getByLabelText("自定义网卡名称")).toBeInTheDocument()
    await user.click(screen.getByRole("combobox", { name: "绑定接口" }))
    await user.click(await screen.findByRole("option", { name: "tun0" }))
    expect(onChange).toHaveBeenLastCalledWith({ bind_interface: "tun0" })
    await user.click(screen.getByRole("combobox", { name: "绑定接口" }))
    await user.click(await screen.findByRole("option", { name: "wlp3s0 (192.168.1.48)" }))
    expect(onChange).toHaveBeenLastCalledWith({ bind_interface: "wlp3s0" })
    await user.click(screen.getByRole("combobox", { name: "绑定接口" }))
    await user.click(await screen.findByRole("option", { name: "未设置" }))
    expect(onChange).toHaveBeenLastCalledWith({})
    await user.click(screen.getByRole("combobox", { name: "绑定接口" }))
    await user.click(await screen.findByRole("option", { name: "eno1" }))
    expect(onChange).toHaveBeenLastCalledWith({ bind_interface: "eno1" })
    await user.click(screen.getByRole("combobox", { name: "绑定接口" }))
    await user.click(await screen.findByRole("option", { name: "手动输入" }))
    fireEvent.change(screen.getByLabelText("自定义网卡名称"), { target: { value: "eth0" } })
    expect(onChange).toHaveBeenLastCalledWith({ bind_interface: "eth0" })
  })
})

describe("inbound hierarchical fields", () => {
  it("hides TLS child fields until TLS is enabled and prunes them when disabled", async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    const view = renderApp(<InboundFormFields fields={tlsFields} object={{ tls: { enabled: true, server_name: "example.com", reality: { enabled: true, private_key: "k" } } }} type="vless" onChange={onChange} />)
    expect(screen.getByLabelText("服务器名称")).toHaveValue("example.com")
    expect(screen.getByLabelText("Reality 私钥")).toHaveValue("k")
    await user.click(screen.getByRole("switch", { name: "启用 TLS" }))
    expect(onChange).toHaveBeenLastCalledWith({})
    view.unmount()

    const off = renderApp(<InboundFormFields fields={tlsFields} object={{}} type="vless" onChange={onChange} />)
    expect(screen.queryByLabelText("服务器名称")).not.toBeInTheDocument()
    expect(screen.queryByLabelText("Reality 私钥")).not.toBeInTheDocument()
    off.unmount()
  })

  it("shows ACME details only after domain is set and keep-alive fields respect disable switch", async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    renderApp(<InboundFormFields fields={tlsFields} object={{ tls: { enabled: true, acme: { domain: ["a.com"], email: "a@b.com" } } }} type="trojan" onChange={onChange} />)
    expect(screen.getByLabelText("ACME 邮箱")).toHaveValue("a@b.com")
    fireEvent.change(screen.getByLabelText("ACME 域名"), { target: { value: "" } })
    expect(onChange).toHaveBeenLastCalledWith({ tls: { enabled: true } })
    cleanup()

    const keep = renderApp(<InboundFormFields fields={[{ path: "disable_tcp_keep_alive", label: "disableTCPKeepAlive", kind: "boolean" }, { path: "tcp_keep_alive", label: "tcpKeepAlive", when: { path: "disable_tcp_keep_alive", falsy: true } }, { path: "tcp_keep_alive_interval", label: "tcpKeepAliveInterval", when: { path: "disable_tcp_keep_alive", falsy: true } }]} object={{ tcp_keep_alive: "5m", tcp_keep_alive_interval: "30s" }} type="mixed" onChange={onChange} />)
    expect(screen.getByLabelText("TCP Keep Alive")).toHaveValue("5m")
    await user.click(screen.getByRole("switch", { name: "禁用 TCP Keep Alive" }))
    expect(onChange).toHaveBeenLastCalledWith({ disable_tcp_keep_alive: true })
    keep.unmount()
  })

  it("prunes nested invisible fields through helpers", () => {
    const pruned = pruneInvisibleFields({
      tls: { enabled: false, server_name: "x", reality: { enabled: true, private_key: "k" } },
      multiplex: { enabled: false, padding: true, brutal: { enabled: true, up_mbps: 100 } },
    }, tlsFields.concat([
      { path: "multiplex.enabled", label: "multiplexEnabled", kind: "boolean" },
      { path: "multiplex.padding", label: "multiplexPadding", kind: "boolean", when: { path: "multiplex.enabled", is: true } },
      { path: "multiplex.brutal.enabled", label: "brutalEnabled", kind: "boolean", when: { path: "multiplex.enabled", is: true } },
      { path: "multiplex.brutal.up_mbps", label: "uploadMbps", kind: "number", when: [{ path: "multiplex.enabled", is: true }, { path: "multiplex.brutal.enabled", is: true }] },
    ]))
    expect(pruned).toEqual({ tls: { enabled: false }, multiplex: { enabled: false } })
    expect(isFieldVisible({ tls: { enabled: true } }, tlsFields[1])).toBe(true)
    expect(applyInboundFieldChange({}, { disable_tcp_keep_alive: true, tcp_keep_alive: "5m" }, "mixed")).toEqual({ disable_tcp_keep_alive: true })
  })
})

describe("inbound ref and network multi fields", () => {
  it("selects detour from inbound tags and network multi options", async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    renderApp(<InboundFormFields fields={[{ path: "detour", label: "detour", kind: "ref", ref: "inbound" }, { path: "network", label: "network", kind: "network-multi" }, { path: "domain_resolver", label: "domainResolver", kind: "ref", ref: "dns-server" }]} object={{}} type="mixed" context={{ inboundTags: ["mixed-in", "tun-in"], dnsServerTags: ["local", "remote"] }} onChange={onChange} />)
    await user.click(screen.getByRole("combobox", { name: "前置入站" }))
    await user.click(await screen.findByRole("option", { name: "mixed-in" }))
    expect(onChange).toHaveBeenLastCalledWith({ detour: "mixed-in" })
    await user.click(screen.getByRole("switch", { name: "网络 tcp" }))
    expect(onChange).toHaveBeenLastCalledWith({ network: ["tcp"] })
    await user.click(screen.getByRole("combobox", { name: "域名解析器" }))
    await user.click(await screen.findByRole("option", { name: "local" }))
    expect(onChange).toHaveBeenLastCalledWith({ domain_resolver: "local" })
  })
})

function UsersHarness() {
  const [object, setObject] = useState<JsonObject>({ users: [{ username: "a", password: "b" }] })
  return <InboundFormFields fields={[{ path: "users", label: "users", kind: "users" }]} object={object} type="mixed" onChange={setObject} />
}

describe("inbound users structured and fallback", () => {
  it("adds structured users and falls back to JSON for invalid arrays", async () => {
    const user = userEvent.setup()
    renderApp(<UsersHarness />)
    expect(screen.getByLabelText("认证用户 1 用户名")).toHaveValue("a")
    await user.click(screen.getByRole("button", { name: "添加用户" }))
    expect(screen.getByLabelText("认证用户 2 用户名")).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText("认证用户 1 密码"), { target: { value: "new" } })
    expect(screen.getByLabelText("认证用户 1 密码")).toHaveValue("new")
    cleanup()

    const onChange = vi.fn()
    const onValidity = vi.fn()
    renderApp(<InboundFormFields fields={[{ path: "users", label: "users", kind: "users" }]} object={{ users: 1 as unknown as never }} type="mixed" onChange={onChange} onFieldValidityChange={onValidity} />)
    const area = screen.getByLabelText("认证用户")
    fireEvent.change(area, { target: { value: "invalid" } })
    expect(onValidity).toHaveBeenLastCalledWith("users", false)
    fireEvent.change(area, { target: { value: '[{"username":"x","password":"y"}]' } })
    expect(onChange).toHaveBeenLastCalledWith({ users: [{ username: "x", password: "y" }] })
    fireEvent.change(area, { target: { value: "" } })
    expect(onChange).toHaveBeenLastCalledWith({})
  })

  it("toggles interface multi selection for TUN filters", async () => {
    const user = userEvent.setup()
    const fetchMock = vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path.includes("/api/network/interfaces")) {
        return Promise.resolve(new Response(JSON.stringify({
          status: "ok",
          data: { interfaces: [{ name: "eth0", ips: ["10.0.0.1"] }, { name: "wlan0", ips: [] }] },
          error: null,
          meta: null,
        }), { status: 200, headers: { "Content-Type": "application/json" } }))
      }
      return Promise.resolve(new Response(JSON.stringify({ status: "ok", data: {}, error: null, meta: null }), { status: 200 }))
    })
    vi.stubGlobal("fetch", fetchMock)
    function InterfaceHarness() {
      const [object, setObject] = useState<JsonObject>({ include_interface: ["custom0"] })
      return <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <InboundFormFields fields={[{ path: "include_interface", label: "includeInterface", kind: "ref", ref: "network-interface-multi" }]} object={object} type="tun" onChange={setObject} />
        <pre>{JSON.stringify(object)}</pre>
      </QueryClientProvider>
    }
    renderApp(<InterfaceHarness />)
    expect(await screen.findByText("eth0 (10.0.0.1)")).toBeInTheDocument()
    await user.click(screen.getByRole("checkbox", { name: /eth0/ }))
    expect(screen.getByText('{"include_interface":["custom0","eth0"]}')).toBeInTheDocument()
    await user.click(screen.getByRole("checkbox", { name: /custom0/ }))
    expect(screen.getByText('{"include_interface":["eth0"]}')).toBeInTheDocument()
  })
})

describe("userSchema", () => {
  it("returns protocol specific keys", () => {
    expect(userSchema("vless")).toEqual(["name", "uuid", "flow"])
    expect(userSchema("mixed")).toEqual(["username", "password"])
    expect(userSchema("unknown")).toEqual(["name", "password"])
  })
})

describe("proxy form helpers", () => {
  it("extracts config and dns tags and groups sections", () => {
    expect(configTags([{ tag: "a" }, { tag: "b" }, { tag: "a" }, null, "x"], "b")).toEqual(["a"])
    expect(configTags(undefined)).toEqual([])
    expect(dnsServerTags({ servers: [{ tag: "local" }, { tag: "" }, {}] })).toEqual(["local"])
    expect(dnsServerTags(null)).toEqual([])
    expect(groupFieldsBySection([
      { path: "a", label: "a", section: "bind" },
      { path: "b", label: "b", section: "bind" },
      { path: "c", label: "c", section: "tcp" },
      { path: "d", label: "d" },
    ]).map((group) => [group.section, group.fields.map((field) => field.path)])).toEqual([
      ["bind", ["a", "b"]],
      ["tcp", ["c"]],
      [undefined, ["d"]],
    ])
  })

  it("covers remaining user schemas and preserves unknown ref values", async () => {
    for (const [type, keys] of [
      ["socks", ["username", "password"]],
      ["http", ["username", "password"]],
      ["naive", ["username", "password"]],
      ["anytls", ["username", "password"]],
      ["trojan", ["name", "password"]],
      ["shadowsocks", ["name", "password"]],
      ["shadowtls", ["name", "password"]],
      ["vmess", ["name", "uuid", "alterId"]],
      ["tuic", ["name", "uuid", "password"]],
      ["hysteria", ["name", "password"]],
      ["hysteria2", ["name", "password"]],
    ] as const) expect(userSchema(type)).toEqual([...keys])
    const user = userEvent.setup()
    function RefHarness() {
      const [object, setObject] = useState<JsonObject>({ detour: "legacy", network: ["tcp"] })
      return <>
        <InboundFormFields fields={[{ path: "detour", label: "detour", kind: "ref", ref: "inbound" }, { path: "network", label: "network", kind: "network-multi" }]} object={object} type="mixed" context={{ inboundTags: ["mixed-in"] }} onChange={setObject} />
        <pre>{JSON.stringify(object)}</pre>
      </>
    }
    renderApp(<RefHarness />)
    await user.click(screen.getByRole("combobox", { name: "前置入站" }))
    expect(await screen.findByRole("option", { name: "legacy" })).toBeInTheDocument()
    await user.click(await screen.findByRole("option", { name: "未设置" }))
    expect(screen.getByText('{"network":["tcp"]}')).toBeInTheDocument()
    await user.click(screen.getByRole("switch", { name: "网络 tcp" }))
    expect(screen.getByText("{}")).toBeInTheDocument()
    await user.click(screen.getByRole("switch", { name: "网络 udp" }))
    expect(screen.getByText('{"network":["udp"]}')).toBeInTheDocument()
  })

  it("edits vmess alterId and removes empty users", async () => {
    const user = userEvent.setup()
    function VmessUsers() {
      const [object, setObject] = useState<JsonObject>({ users: [{ name: "n", uuid: "u", alterId: 1 }] })
      return <>
        <InboundFormFields fields={[{ path: "users", label: "users", kind: "users" }]} object={object} type="vmess" onChange={setObject} />
        <pre>{JSON.stringify(object)}</pre>
      </>
    }
    renderApp(<VmessUsers />)
    fireEvent.change(screen.getByLabelText("认证用户 1 Alter ID"), { target: { value: "2" } })
    expect(screen.getByText('{"users":[{"name":"n","uuid":"u","alterId":2}]}')).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText("认证用户 1 名称"), { target: { value: "" } })
    expect(screen.getByText('{"users":[{"uuid":"u","alterId":2}]}')).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "删除用户" }))
    expect(screen.getByText("{}")).toBeInTheDocument()
    cleanup()

    renderApp(<InboundFormFields fields={[{ path: "users", label: "users", kind: "users" }]} object={{}} type="mixed" onChange={vi.fn()} />)
    expect(screen.getByText("暂无用户，可点击添加。")).toBeInTheDocument()
  })

  it("shows empty interface multi state", async () => {
    const fetchMock = vi.fn(() => Promise.resolve(new Response(JSON.stringify({
      status: "ok", data: { interfaces: [] }, error: null, meta: null,
    }), { status: 200, headers: { "Content-Type": "application/json" } })))
    vi.stubGlobal("fetch", fetchMock)
    renderApp(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><InboundFormFields fields={[{ path: "include_interface", label: "includeInterface", kind: "ref", ref: "network-interface-multi" }]} object={{}} type="tun" onChange={vi.fn()} /></QueryClientProvider>)
    expect(await screen.findByText("—")).toBeInTheDocument()
  })

  it("covers dns/outbound refs, invalid users, and sparse field branches", async () => {
    const user = userEvent.setup()
    function MultiRefHarness() {
      const [object, setObject] = useState<JsonObject>({
        detour: "out-legacy",
        domain_resolver: "dns-legacy",
        custom_ref: "kept",
        ports: [1, 2],
        count: 3,
      })
      return <>
        <ProxyFormFields
          fields={[
            { path: "detour", label: "detour", kind: "ref", ref: "outbound" },
            { path: "domain_resolver", label: "domainResolver", kind: "ref", ref: "dns-server" },
            { path: "custom_ref", label: "method", kind: "ref", options: ["a", "b"] },
            { path: "ports", label: "listenPort", kind: "number-list" },
            { path: "count", label: "mtu", kind: "number" },
            { path: "note", label: "password", kind: "text", section: "missing-section" },
          ]}
          object={object}
          namespace="proxy.inbound"
          context={{ outboundTags: ["proxy"], dnsServerTags: ["local"] }}
          onChange={setObject}
        />
        <pre data-testid="object">{JSON.stringify(object)}</pre>
      </>
    }
    renderApp(<MultiRefHarness />)
    await user.click(screen.getByRole("combobox", { name: "前置入站" }))
    expect(await screen.findByRole("option", { name: "out-legacy" })).toBeInTheDocument()
    await user.click(await screen.findByRole("option", { name: "proxy" }))
    await user.click(screen.getByRole("combobox", { name: "域名解析器" }))
    expect(await screen.findByRole("option", { name: "dns-legacy" })).toBeInTheDocument()
    await user.click(await screen.findByRole("option", { name: "local" }))
    await user.click(screen.getByRole("combobox", { name: "加密方法" }))
    await user.click(await screen.findByRole("option", { name: "a" }))
    fireEvent.change(screen.getByLabelText("监听端口"), { target: { value: "bad" } })
    expect(screen.getByTestId("object")).toHaveTextContent('{"detour":"proxy","domain_resolver":"local","custom_ref":"a","ports":[1,2],"count":3}')
    fireEvent.change(screen.getByLabelText("监听端口"), { target: { value: "" } })
    fireEvent.change(screen.getByLabelText("MTU"), { target: { value: "" } })
    expect(screen.getByTestId("object")).toHaveTextContent('{"detour":"proxy","domain_resolver":"local","custom_ref":"a"}')
    cleanup()

    function InvalidUsers() {
      const [object, setObject] = useState<JsonObject>({ users: "broken" as unknown as never })
      const [valid, setValid] = useState(true)
      return <>
        <span data-testid="validity">{valid ? "valid" : "invalid"}</span>
        <InboundFormFields fields={[{ path: "users", label: "users", kind: "users" }]} object={object} type="mixed" onChange={setObject} onFieldValidityChange={(_path, next) => setValid(next)} />
        <pre data-testid="users-object">{JSON.stringify(object)}</pre>
      </>
    }
    renderApp(<InvalidUsers />)
    const textarea = screen.getByLabelText("认证用户")
    fireEvent.change(textarea, { target: { value: "{" } })
    expect(screen.getByTestId("validity")).toHaveTextContent("invalid")
    fireEvent.change(textarea, { target: { value: '[{"username":"a","password":"b"}]' } })
    expect(screen.getByTestId("validity")).toHaveTextContent("valid")
    expect(screen.getByTestId("users-object")).toHaveTextContent('{"users":[{"username":"a","password":"b"}]}')
    cleanup()

    // non-object array item keeps JSON mode invalid
    function BadArrayUsers() {
      const [object, setObject] = useState<JsonObject>({ users: [1] as unknown as never })
      const [valid, setValid] = useState(true)
      return <>
        <span data-testid="validity2">{valid ? "valid" : "invalid"}</span>
        <InboundFormFields fields={[{ path: "users", label: "users", kind: "users" }]} object={object} type="mixed" onChange={setObject} onFieldValidityChange={(_path, next) => setValid(next)} />
      </>
    }
    renderApp(<BadArrayUsers />)
    fireEvent.change(screen.getByLabelText("认证用户"), { target: { value: "" } })
    expect(screen.getByTestId("validity2")).toHaveTextContent("valid")
    cleanup()

    function AlterIdEmpty() {
      const [object, setObject] = useState<JsonObject>({ users: [{ name: "n", uuid: "u", alterId: 1 }] })
      return <>
        <InboundFormFields fields={[{ path: "users", label: "users", kind: "users" }]} object={object} type="vmess" onChange={setObject} />
        <pre>{JSON.stringify(object)}</pre>
      </>
    }
    renderApp(<AlterIdEmpty />)
    fireEvent.change(screen.getByLabelText("认证用户 1 Alter ID"), { target: { value: "" } })
    expect(screen.getByText('{"users":[{"name":"n","uuid":"u"}]}')).toBeInTheDocument()
  })
})
