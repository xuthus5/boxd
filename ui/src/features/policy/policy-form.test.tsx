import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { useState } from "react"
import { cleanup, fireEvent, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { PolicyFormFields } from "@/features/policy/policy-form-fields"
import { i18n } from "@/i18n"
import {
  cloneJsonObject,
  getPolicyPath,
  isJsonObject,
  moveItem,
  groupPolicyFieldsBySection,
  isPolicyFieldVisible,
  policyConfigTags,
  policyDNSServerTags,
  policyRuleSetTags,
  pruneInvisiblePolicyFields,
  setPolicyPath,
  visiblePolicyFields,
  type JsonObject,
  type PolicyFieldSpec,
} from "@/features/policy/policy-form-model"
import { installMockAPI } from "@/test/mock-api"
import { renderApp } from "@/test/render"

beforeEach(() => {
  installMockAPI()
})

afterEach(() => {
  vi.unstubAllGlobals()
  cleanup()
})

describe("policy form JSON helpers", () => {
  it("recognizes JSON objects", () => {
    expect(isJsonObject({ action: "route" })).toBe(true)
    expect(isJsonObject([])).toBe(false)
    expect(isJsonObject(null)).toBe(false)
    expect(isJsonObject(undefined)).toBe(false)
  })

  it("reads nested object paths", () => {
    expect(getPolicyPath({ action: { server: "dns-remote" } }, "action.server")).toBe("dns-remote")
    expect(getPolicyPath({ action: [] }, "action.server")).toBeUndefined()
    expect(getPolicyPath({}, "action.server")).toBeUndefined()
  })

  it("writes nested paths without mutating the source", () => {
    const source = { action: { outbound: "proxy" } }
    const next = setPolicyPath(source, "action.server", "dns-remote")

    expect(next).toEqual({ action: { outbound: "proxy", server: "dns-remote" } })
    expect(source).toEqual({ action: { outbound: "proxy" } })
    expect(next).not.toBe(source)
    expect(next.action).not.toBe(source.action)
    expect(setPolicyPath({}, "action.server", "dns-remote")).toEqual({ action: { server: "dns-remote" } })
    expect(setPolicyPath({ action: [] }, "action.server", "dns-remote")).toEqual({ action: { server: "dns-remote" } })
  })

  it("prunes empty parents when nested values are removed", () => {
    expect(setPolicyPath({ action: { server: "dns-remote" } }, "action.server", undefined)).toEqual({})
    expect(setPolicyPath({ action: { server: "dns-remote", strategy: "prefer_ipv4" } }, "action.server", undefined)).toEqual({ action: { strategy: "prefer_ipv4" } })
    expect(setPolicyPath({ tag: "rule" }, "tag", undefined)).toEqual({})
  })

  it("moves items immutably and leaves invalid moves unchanged", () => {
    const source = ["a", "b", "c"] as const

    expect(moveItem(source, 1, -1)).toEqual(["b", "a", "c"])
    expect(moveItem(source, 1, 1)).toEqual(["a", "c", "b"])
    expect(moveItem(["a", "b"], 0, -1)).toEqual(["a", "b"])
    expect(moveItem(["a", "b"], 1, 1)).toEqual(["a", "b"])
    expect(moveItem(["a", "b"], -1, 1)).toEqual(["a", "b"])
    expect(source).toEqual(["a", "b", "c"])
  })

  it("deep clones JSON objects", () => {
    const source = { rules: [{ action: "route" }], options: { enabled: true } }
    const clone = cloneJsonObject(source)

    expect(clone).toEqual(source)
    expect(clone).not.toBe(source)
    expect(clone.rules).not.toBe(source.rules)
    expect(clone.options).not.toBe(source.options)
  })
})

const namespace = "policy.dns"
const label = (name: string) => i18n.t(`${namespace}.${name}`)

function renderFields(fields: readonly PolicyFieldSpec[], object: JsonObject = {}) {
  const onChange = vi.fn()
  renderApp(<PolicyFormFields fields={fields} object={object} namespace={namespace} onChange={onChange} />)
  return onChange
}

function SelectHarness() {
  const [object, setObject] = useState<JsonObject>({})
  const [valid, setValid] = useState(true)
  const [, setRender] = useState(0)
  return <>
    <button onClick={() => setRender((value) => value + 1)}>Rerender</button>
    <output>{String(object.action)}</output>
    <output aria-label="select validity">{valid ? "valid" : "invalid"}</output>
    <PolicyFormFields
      fields={[{ path: "action", label: "action", kind: "select", options: ["__unset__", "reject"], required: true }]}
      object={object}
      namespace={namespace}
      onChange={setObject}
      onFieldValidityChange={(_path, next) => setValid(next)}
    />
  </>
}

function StructuredHarness() {
  const [object, setObject] = useState<JsonObject>({ headers: { X: "old" }, tag: "old" })
  const [revision, setRevision] = useState(0)
  const [valid, setValid] = useState(true)
  const [, setRender] = useState(0)
  return <>
    <button onClick={() => setRender((value) => value + 1)}>Parent rerender</button>
    <button onClick={() => { setObject((value) => ({ ...value, tag: "changed" })); setRevision((value) => value + 1) }}>Advanced JSON update</button>
    <button onClick={() => setObject((value) => ({ ...value, headers: { X: "new" } }))}>External field update</button>
    <output>{valid ? "valid" : "invalid"}</output>
    <PolicyFormFields
      fields={[{ path: "headers", label: "headers", kind: "json-object" }]}
      object={object}
      namespace={namespace}
      revision={revision}
      onChange={setObject}
      onFieldValidityChange={(_path, next) => setValid(next)}
    />
  </>
}

describe("policy form field conversions", () => {
  it("updates text, textarea, numbers, booleans, and required controls", async () => {
    const user = userEvent.setup()
    const textChange = renderFields([{ path: "tag", label: "tag", required: true }])
    const text = screen.getByLabelText(label("tag"))
    expect(text).toBeRequired()
    fireEvent.change(text, { target: { value: "rule-1" } })
    expect(textChange).toHaveBeenLastCalledWith({ tag: "rule-1" })
    cleanup()

    const areaChange = renderFields([{ path: "description", label: "description", kind: "textarea" }])
    fireEvent.change(screen.getByLabelText(label("description")), { target: { value: "line 1\nline 2" } })
    expect(areaChange).toHaveBeenLastCalledWith({ description: "line 1\nline 2" })
    cleanup()

    const numberChange = renderFields([{ path: "port", label: "port", kind: "number" }], { port: 53 })
    fireEvent.change(screen.getByLabelText(label("port")), { target: { value: "443" } })
    expect(numberChange).toHaveBeenLastCalledWith({ port: 443 })
    cleanup()

    const emptyNumberChange = renderFields([{ path: "port", label: "port", kind: "number" }], { port: 53 })
    fireEvent.change(screen.getByLabelText(label("port")), { target: { value: "" } })
    expect(emptyNumberChange).toHaveBeenLastCalledWith({})
    cleanup()

    const booleanChange = renderFields([{ path: "invert", label: "invert", kind: "boolean" }], { invert: true })
    await user.click(screen.getByRole("switch", { name: label("invert") }))
    expect(booleanChange).toHaveBeenLastCalledWith({ invert: false })
  })

  it("parses string and numeric lists and rejects non-finite values", () => {
    const listChange = renderFields([{ path: "domain", label: "domain", kind: "list" }])
    fireEvent.change(screen.getByLabelText(label("domain")), { target: { value: "example.com, example.org\nexample.net" } })
    expect(listChange).toHaveBeenLastCalledWith({ domain: ["example.com", "example.org", "example.net"] })
    cleanup()

    const emptyListChange = renderFields([{ path: "domain", label: "domain", kind: "list" }], { domain: ["example.com"] })
    fireEvent.change(screen.getByLabelText(label("domain")), { target: { value: "" } })
    expect(emptyListChange).toHaveBeenLastCalledWith({})
    cleanup()

    const numberListChange = renderFields([{ path: "port", label: "port", kind: "number-list" }])
    fireEvent.change(screen.getByLabelText(label("port")), { target: { value: "53, 853" } })
    expect(numberListChange).toHaveBeenLastCalledWith({ port: [53, 853] })
    fireEvent.change(screen.getByLabelText(label("port")), { target: { value: "1e309" } })
    expect(numberListChange).toHaveBeenCalledTimes(1)
    cleanup()

    const emptyNumberListChange = renderFields([{ path: "port", label: "port", kind: "number-list" }], { port: [53] })
    fireEvent.change(screen.getByLabelText(label("port")), { target: { value: "" } })
    expect(emptyNumberListChange).toHaveBeenLastCalledWith({})
  })

  it("uses null for required Base UI Select state without colliding with option values", async () => {
    const user = userEvent.setup()
    renderApp(<SelectHarness />)

    await user.click(screen.getByRole("button", { name: "Rerender" }))
    const select = screen.getByRole("combobox", { name: label("action") })
    expect(select).toHaveAttribute("aria-invalid", "true")
    expect(screen.getByLabelText("select validity")).toHaveTextContent("invalid")

    await user.click(select)
    await user.click(await screen.findByRole("option", { name: "__unset__" }))
    expect(screen.getByText("__unset__", { selector: "output" })).toBeInTheDocument()
    expect(select).toHaveAttribute("aria-invalid", "false")
    await user.click(select)
    await user.click(await screen.findByRole("option", { name: label("notSet") }))
    expect(screen.getByText("undefined", { selector: "output" })).toBeInTheDocument()
    expect(select).toHaveAttribute("aria-invalid", "true")
  })

  it("uses custom transforms and falls back only when they return undefined", () => {
    const onChange = vi.fn()
    const onValidity = vi.fn()
    const transform = vi.fn((_object, _field, raw: string) => raw === "ignore" ? null : raw === "fallback" ? undefined : { transformed: raw })
    renderApp(<PolicyFormFields fields={[{ path: "tag", label: "tag" }]} object={{}} namespace={namespace}
      onChange={onChange} onFieldValidityChange={onValidity} transformField={transform} />)
    const input = screen.getByLabelText(label("tag"))

    fireEvent.change(input, { target: { value: "custom" } })
    expect(onChange).toHaveBeenLastCalledWith({ transformed: "custom" })
    fireEvent.change(input, { target: { value: "ignore" } })
    expect(onChange).toHaveBeenCalledTimes(1)
    expect(input).toHaveAttribute("aria-invalid", "true")
    expect(onValidity).toHaveBeenLastCalledWith("tag", false)
    fireEvent.change(input, { target: { value: "fallback" } })
    expect(onChange).toHaveBeenLastCalledWith({ tag: "fallback" })
    expect(input).toHaveAttribute("aria-invalid", "false")
    expect(onValidity).toHaveBeenLastCalledWith("tag", true)
  })

  it("validates JSON objects and arrays and clears invalid state", () => {
    const onChange = vi.fn()
    const onValidity = vi.fn()
    const view = renderApp(<PolicyFormFields
      fields={[
        { path: "headers", label: "headers", kind: "json-object" },
        { path: "rules", label: "rules", kind: "json-array" },
      ]}
      object={{}}
      namespace={namespace}
      onChange={onChange}
      onFieldValidityChange={onValidity}
    />)

    fireEvent.change(screen.getByLabelText(label("headers")), { target: { value: "{}" } })
    expect(onChange).toHaveBeenLastCalledWith({ headers: {} })
    fireEvent.change(screen.getByLabelText(label("headers")), { target: { value: "[]" } })
    expect(screen.getByLabelText(label("headers"))).toHaveAttribute("aria-invalid", "true")
    expect(onValidity).toHaveBeenLastCalledWith("headers", false)
    fireEvent.change(screen.getByLabelText(label("headers")), { target: { value: "" } })
    expect(onChange).toHaveBeenLastCalledWith({})
    expect(onValidity).toHaveBeenLastCalledWith("headers", true)

    fireEvent.change(screen.getByLabelText(label("rules")), { target: { value: '[1,{"action":"route"}]' } })
    expect(onChange).toHaveBeenLastCalledWith({ rules: [1, { action: "route" }] })
    fireEvent.change(screen.getByLabelText(label("rules")), { target: { value: "{}" } })
    expect(onValidity).toHaveBeenLastCalledWith("rules", false)
    view.unmount()
    expect(onValidity).toHaveBeenLastCalledWith("rules", true)
  })

  it("synchronizes structured drafts after external and Advanced JSON updates", async () => {
    const user = userEvent.setup()
    renderApp(<StructuredHarness />)
    const input = screen.getByLabelText(label("headers"))
    expect(input).toHaveValue(JSON.stringify({ X: "old" }, null, 2))

    fireEvent.change(input, { target: { value: "invalid" } })
    await user.click(screen.getByRole("button", { name: "Parent rerender" }))
    expect(screen.getByText("invalid", { selector: "output" })).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "Advanced JSON update" }))
    expect(input).toHaveValue(JSON.stringify({ X: "old" }, null, 2))
    expect(screen.getByText("valid", { selector: "output" })).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "External field update" }))
    expect(input).toHaveValue(JSON.stringify({ X: "new" }, null, 2))
  })
})


function renderWithQuery(ui: React.ReactElement) {
  return renderApp(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>{ui}</QueryClientProvider>)
}

describe("policy form hierarchy and refs", () => {
  it("renders section titles and hierarchical fields", () => {
    renderApp(<PolicyFormFields
      fields={[
        { path: "tls_fragment", label: "tlsFragment", kind: "boolean", section: "action" },
        { path: "tls_fragment_fallback_delay", label: "tlsFragmentFallbackDelay", section: "action", when: { path: "tls_fragment", is: true } },
      ]}
      object={{ tls_fragment: true, tls_fragment_fallback_delay: "500ms" }}
      namespace="policy.route"
      onChange={vi.fn()}
    />)
    expect(screen.getByText("执行动作")).toBeInTheDocument()
    expect(screen.getByLabelText("TLS 分片回退延迟")).toBeInTheDocument()
    cleanup()
    renderApp(<PolicyFormFields
      fields={[
        { path: "tls_fragment", label: "tlsFragment", kind: "boolean", section: "action" },
        { path: "tls_fragment_fallback_delay", label: "tlsFragmentFallbackDelay", section: "action", when: { path: "tls_fragment", is: true } },
      ]}
      object={{ tls_fragment: false }}
      namespace="policy.route"
      onChange={vi.fn()}
    />)
    expect(screen.queryByLabelText("TLS 分片回退延迟")).not.toBeInTheDocument()
  })

  it("selects outbound refs and falls back to text input without options", async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    renderApp(<PolicyFormFields
      fields={[{ path: "outbound", label: "outbound", kind: "ref", ref: "outbound", required: true }]}
      object={{}}
      namespace="policy.route"
      context={{ outboundTags: ["proxy", "direct"] }}
      onChange={onChange}
    />)
    await user.click(screen.getByRole("combobox", { name: "目标出站" }))
    await user.click(await screen.findByRole("option", { name: "proxy" }))
    expect(onChange).toHaveBeenLastCalledWith({ outbound: "proxy" })
    cleanup()

    const textChange = vi.fn()
    renderApp(<PolicyFormFields
      fields={[{ path: "server", label: "resolveServer", kind: "ref", ref: "dns-server", required: true }]}
      object={{}}
      namespace="policy.route"
      onChange={textChange}
    />)
    fireEvent.change(screen.getByLabelText("解析服务器"), { target: { value: "dns-remote" } })
    expect(textChange).toHaveBeenLastCalledWith({ server: "dns-remote" })
  })

  it("toggles network multi values and selects network interfaces", async () => {
    const user = userEvent.setup()
    function Harness() {
      const [object, setObject] = useState<JsonObject>({})
      return <>
        <output aria-label="form state">{JSON.stringify(object)}</output>
        <PolicyFormFields
          fields={[
            { path: "network", label: "network", kind: "network-multi" },
            { path: "default_interface", label: "defaultInterface", kind: "network-interface" },
          ]}
          object={object}
          namespace="policy.route"
          onChange={setObject}
        />
      </>
    }
    renderWithQuery(<Harness />)
    await user.click(screen.getByRole("switch", { name: /tcp/ }))
    expect(screen.getByLabelText("form state")).toHaveTextContent('"network":["tcp"]')
    await user.click(screen.getByRole("switch", { name: /udp/ }))
    expect(screen.getByLabelText("form state")).toHaveTextContent('"network":["tcp","udp"]')
    await user.click(screen.getByRole("switch", { name: /tcp/ }))
    expect(screen.getByLabelText("form state")).toHaveTextContent('"network":["udp"]')

    await user.click(await screen.findByRole("combobox", { name: "默认接口" }))
    await user.click(await screen.findByRole("option", { name: /eth0/ }))
    expect(screen.getByLabelText("form state")).toHaveTextContent('"default_interface":"eth0"')

    await user.click(screen.getByRole("combobox", { name: "默认接口" }))
    await user.click(await screen.findByRole("option", { name: "手动输入" }))
    fireEvent.change(screen.getByLabelText("自定义网卡名称"), { target: { value: "wg0" } })
    expect(screen.getByLabelText("form state")).toHaveTextContent('"default_interface":"wg0"')

    await user.click(screen.getByRole("combobox", { name: "默认接口" }))
    await user.click(await screen.findByRole("option", { name: "未设置" }))
    expect(screen.getByLabelText("form state")).not.toHaveTextContent("default_interface")
  })

  it("covers visibility prune helpers", () => {
    const fields = [
      { path: "tls_fragment", label: "tlsFragment", kind: "boolean" as const },
      { path: "tls_fragment_fallback_delay", label: "delay", when: { path: "tls_fragment", is: true as const } },
    ]
    expect(visiblePolicyFields(fields, { tls_fragment: false }).map((field) => field.path)).toEqual(["tls_fragment"])
    expect(pruneInvisiblePolicyFields({ tls_fragment: false, tls_fragment_fallback_delay: "1s" }, fields)).toEqual({ tls_fragment: false })
    expect(isPolicyFieldVisible({ mode: "x" }, { path: "child", label: "c", when: { path: "mode", is: ["a", "b"] } })).toBe(false)
    expect(isPolicyFieldVisible({ mode: [] }, { path: "child", label: "c", when: { path: "mode", falsy: true } })).toBe(true)
  })
})


describe("policy form ref variants and helpers", () => {
  it("supports inbound/rule-set refs, network-interface ref kind, and leading content", async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    renderWithQuery(<PolicyFormFields
      fields={[
        { path: "inbound", label: "inbound", kind: "ref", ref: "inbound" },
        { path: "rule_set", label: "ruleSet", kind: "ref", ref: "rule-set" },
        { path: "bind_interface", label: "defaultInterface", kind: "ref", ref: "network-interface" },
        { path: "notes", label: "description", kind: "textarea" },
      ]}
      object={{ inbound: "legacy-in" }}
      namespace="policy.route"
      context={{ inboundTags: ["mixed-in"], ruleSetTags: ["geo"] }}
      leading={<div>leading-slot</div>}
      onChange={onChange}
    />)
    expect(screen.getByText("leading-slot")).toBeInTheDocument()
    await user.click(screen.getByRole("combobox", { name: "入站" }))
    await user.click(await screen.findByRole("option", { name: /mixed-in/ }))
    expect(onChange).toHaveBeenLastCalledWith({ inbound: "mixed-in" })
    await user.click(screen.getByRole("combobox", { name: "规则集" }))
    await user.click(await screen.findByRole("option", { name: /geo/ }))
    expect(onChange).toHaveBeenLastCalledWith({ inbound: "legacy-in", rule_set: "geo" })
    await user.click(await screen.findByRole("combobox", { name: "默认接口" }))
    await user.click(await screen.findByRole("option", { name: /wlan0/ }))
    expect(onChange).toHaveBeenLastCalledWith({ inbound: "legacy-in", bind_interface: "wlan0" })
  })

  it("covers tag helpers and section grouping", () => {
    expect(policyConfigTags([{ tag: "a" }, { tag: "a" }, { tag: "b" }, null, "x"], "b")).toEqual(["a"])
    expect(policyConfigTags("bad")).toEqual([])
    expect(policyDNSServerTags({ servers: [{ tag: "dns" }] })).toEqual(["dns"])
    expect(policyDNSServerTags([])).toEqual([])
    expect(policyRuleSetTags({ rule_set: [{ tag: "geo" }] })).toEqual(["geo"])
    expect(policyRuleSetTags(null)).toEqual([])
    const fields = [
      { path: "a", label: "a", section: "basic" },
      { path: "b", label: "b", section: "basic" },
      { path: "c", label: "c", section: "action" },
      { path: "d", label: "d" },
    ]
    expect(groupPolicyFieldsBySection(fields).map((group) => [group.section, group.fields.length])).toEqual([
      ["basic", 2], ["action", 1], [undefined, 1],
    ])
    expect(isPolicyFieldVisible({ enabled: false }, { path: "child", label: "c", when: { path: "enabled" } })).toBe(false)
    expect(isPolicyFieldVisible({ mode: "a" }, { path: "child", label: "c", when: { path: "mode", is: ["a", "b"] } })).toBe(true)
    expect(isPolicyFieldVisible({ mode: "" }, { path: "child", label: "c", when: { path: "mode", falsy: true } })).toBe(true)
  })
})

describe("policy form interface revision sync", () => {
  it("revision remounts network interface mode for unknown values", async () => {
    const user = userEvent.setup()
    function Harness() {
      const [object, setObject] = useState<JsonObject>({ default_interface: "custom0" })
      const [revision, setRevision] = useState(0)
      return <>
        <button onClick={() => { setObject({ default_interface: "eth0" }); setRevision((value) => value + 1) }}>Sync</button>
        <output aria-label="form state">{JSON.stringify(object)}</output>
        <PolicyFormFields
          fields={[{ path: "default_interface", label: "defaultInterface", kind: "network-interface" }]}
          object={object}
          namespace="policy.route"
          revision={revision}
          onChange={setObject}
        />
      </>
    }
    renderWithQuery(<Harness />)
    expect(screen.getByLabelText("自定义网卡名称")).toHaveValue("custom0")
    await user.click(screen.getByRole("button", { name: "Sync" }))
    expect(screen.queryByLabelText("自定义网卡名称")).not.toBeInTheDocument()
    expect(screen.getByLabelText("form state")).toHaveTextContent('"default_interface":"eth0"')
  })
})

describe("policy form ref clear and unknown option", () => {
  it("clears ref values with not-set option and keeps unknown current values", async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    renderApp(<PolicyFormFields
      fields={[{ path: "outbound", label: "outbound", kind: "ref", ref: "outbound", required: true }]}
      object={{ outbound: "legacy" }}
      namespace="policy.route"
      context={{ outboundTags: ["proxy"] }}
      onChange={onChange}
    />)
    // unknown current value remains selectable
    await user.click(screen.getByRole("combobox", { name: "目标出站" }))
    expect(await screen.findByRole("option", { name: "legacy" })).toBeInTheDocument()
    await user.click(await screen.findByRole("option", { name: "未设置" }))
    expect(onChange).toHaveBeenLastCalledWith({})
  })
})

describe("policy form multi refs and ip version select", () => {
  it("ref-multi inbound and rule-set selections store string arrays", async () => {
    const user = userEvent.setup()
    function Harness() {
      const [object, setObject] = useState<JsonObject>({})
      return <>
        <output aria-label="form state">{JSON.stringify(object)}</output>
        <PolicyFormFields
          fields={[
            { path: "inbound", label: "inbound", kind: "ref-multi", ref: "inbound" },
            { path: "rule_set", label: "ruleSet", kind: "ref-multi", ref: "rule-set" },
            { path: "ip_version", label: "ipVersion", kind: "select", options: ["4", "6"] },
          ]}
          object={object}
          namespace="policy.route"
          context={{ inboundTags: ["mixed-in", "tun-in"], ruleSetTags: ["geo", "cn"] }}
          onChange={setObject}
        />
      </>
    }
    renderApp(<Harness />)
    await user.click(screen.getByLabelText("入站 mixed-in"))
    expect(screen.getByLabelText("form state")).toHaveTextContent('"inbound":["mixed-in"]')
    await user.click(screen.getByLabelText("规则集 geo"))
    expect(screen.getByLabelText("form state")).toHaveTextContent('"rule_set":["geo"]')
    await user.click(screen.getByRole("combobox", { name: "IP 版本" }))
    await user.click(await screen.findByRole("option", { name: "4" }))
    expect(screen.getByLabelText("form state")).toHaveTextContent('"ip_version":4')
  })
})


describe("policy form ref-multi empty and numeric select branches", () => {
  it("shows empty multi-ref hint and clears values", async () => {
    const user = userEvent.setup()
    function Harness() {
      const [object, setObject] = useState<JsonObject>({ inbound: ["legacy"], rule_set: ["old"] })
      return <>
        <output aria-label="form state">{JSON.stringify(object)}</output>
        <PolicyFormFields
          fields={[
            { path: "inbound", label: "inbound", kind: "ref-multi", ref: "inbound" },
            { path: "rule_set", label: "ruleSet", kind: "ref-multi", ref: "rule-set" },
            { path: "ip_version", label: "ipVersion", kind: "select", options: ["4", "6"] },
            { path: "strategy", label: "defaultDomainResolverStrategy", kind: "select", options: ["prefer_ipv4", "prefer_ipv6"] },
          ]}
          object={object}
          namespace="policy.route"
          context={{ inboundTags: [], ruleSetTags: ["geo"] }}
          onChange={setObject}
        />
      </>
    }
    renderApp(<Harness />)
    // unknown current inbound still listed for uncheck
    expect(screen.getByLabelText("入站 legacy")).toBeInTheDocument()
    await user.click(screen.getByLabelText("入站 legacy"))
    expect(screen.getByLabelText("form state")).not.toHaveTextContent("inbound")
    await user.click(screen.getByLabelText("规则集 geo"))
    expect(screen.getByLabelText("form state")).toHaveTextContent('"rule_set":["old","geo"]')
    await user.click(screen.getByLabelText("规则集 old"))
    expect(screen.getByLabelText("form state")).toHaveTextContent('"rule_set":["geo"]')
    // string select path (non-numeric)
    await user.click(screen.getByRole("combobox", { name: "默认域名解析策略" }))
    await user.click(await screen.findByRole("option", { name: "prefer_ipv6" }))
    expect(screen.getByLabelText("form state")).toHaveTextContent('"strategy":"prefer_ipv6"')
    // clear numeric select
    await user.click(screen.getByRole("combobox", { name: "IP 版本" }))
    await user.click(await screen.findByRole("option", { name: "6" }))
    expect(screen.getByLabelText("form state")).toHaveTextContent('"ip_version":6')
    await user.click(screen.getByRole("combobox", { name: "IP 版本" }))
    await user.click(await screen.findByRole("option", { name: "未设置" }))
    expect(screen.getByLabelText("form state")).not.toHaveTextContent("ip_version")
  })

  it("renders empty multi-ref placeholder without context options", () => {
    renderApp(<PolicyFormFields
      fields={[{ path: "inbound", label: "inbound", kind: "ref-multi", ref: "inbound" }]}
      object={{}}
      namespace="policy.route"
      onChange={vi.fn()}
    />)
    expect(screen.getByText("暂无可选项，可先在配置中添加。")).toBeInTheDocument()
  })
})

describe("policy form select transform and multi-ref variants", () => {
  it("covers select transform null and ref-multi outbound/dns", async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    const transform = vi.fn((_object: JsonObject, field: PolicyFieldSpec, raw: string) => {
      if (field.path === "ip_version" && raw === "4") return null
      if (field.path === "ip_version" && raw === "6") return undefined
      return undefined
    })
    renderApp(<PolicyFormFields
      fields={[
        { path: "ip_version", label: "ipVersion", kind: "select", options: ["4", "6"] },
        { path: "outbound", label: "outbound", kind: "ref-multi", ref: "outbound" },
        { path: "server", label: "resolveServer", kind: "ref-multi", ref: "dns-server" },
      ]}
      object={{}}
      namespace="policy.route"
      context={{ outboundTags: ["proxy"], dnsServerTags: ["dns-remote"] }}
      transformField={transform}
      onChange={onChange}
    />)
    await user.click(screen.getByRole("combobox", { name: "IP 版本" }))
    await user.click(await screen.findByRole("option", { name: "4" }))
    expect(onChange).not.toHaveBeenCalled()
    await user.click(screen.getByRole("combobox", { name: "IP 版本" }))
    await user.click(await screen.findByRole("option", { name: "6" }))
    expect(onChange).toHaveBeenLastCalledWith({ ip_version: 6 })
    await user.click(screen.getByLabelText("目标出站 proxy"))
    expect(onChange).toHaveBeenLastCalledWith({ outbound: ["proxy"] })
    await user.click(screen.getByLabelText("解析服务器 dns-remote"))
    expect(onChange).toHaveBeenLastCalledWith({ server: ["dns-remote"] })
  })
})

describe("policy form remaining select/multi branches", () => {
  it("select transform returns object and multi-ref option refs", async () => {
    const user = userEvent.setup()
    function Harness() {
      const [object, setObject] = useState<JsonObject>({})
      return <>
        <output aria-label="form state">{JSON.stringify(object)}</output>
        <PolicyFormFields
          fields={[
            { path: "ip_version", label: "ipVersion", kind: "select", options: ["4", "6"] },
            { path: "tags", label: "ruleSet", kind: "ref-multi", options: ["a", "b"] },
          ]}
          object={object}
          namespace="policy.route"
          transformField={(current, field, raw) => field.path === "ip_version" && raw === "4"
            ? { ...current, ip_version: 4, marked: true }
            : undefined}
          onChange={setObject}
        />
      </>
    }
    renderApp(<Harness />)
    await user.click(screen.getByRole("combobox", { name: "IP 版本" }))
    await user.click(await screen.findByRole("option", { name: "4" }))
    expect(screen.getByLabelText("form state")).toHaveTextContent('"ip_version":4')
    expect(screen.getByLabelText("form state")).toHaveTextContent('"marked":true')
    await user.click(screen.getByLabelText("规则集 a"))
    expect(screen.getByLabelText("form state")).toHaveTextContent('"tags":["a"]')
    await user.click(screen.getByLabelText("规则集 a"))
    expect(screen.getByLabelText("form state")).not.toHaveTextContent("tags")
  })
})

describe("policy form multi-ref options-only branch", () => {
  it("covers multi-ref field.options without ref and empty clear", async () => {
    const user = userEvent.setup()
    function Harness() {
      const [object, setObject] = useState<JsonObject>({ tags: ["b"] })
      return <>
        <output aria-label="form state">{JSON.stringify(object)}</output>
        <PolicyFormFields
          fields={[{ path: "tags", label: "ruleSet", kind: "ref-multi", options: ["a", "b"] }]}
          object={object}
          namespace="policy.route"
          onChange={setObject}
        />
      </>
    }
    renderApp(<Harness />)
    await user.click(screen.getByLabelText("规则集 a"))
    expect(screen.getByLabelText("form state")).toHaveTextContent('"tags":["b","a"]')
    await user.click(screen.getByLabelText("规则集 b"))
    await user.click(screen.getByLabelText("规则集 a"))
    expect(screen.getByLabelText("form state")).not.toHaveTextContent("tags")
  })
})

describe("policy form select clear path", () => {
  it("clears non-numeric select and keeps unknown multi values", async () => {
    const user = userEvent.setup()
    function Harness() {
      const [object, setObject] = useState<JsonObject>({ strategy: "prefer_ipv4", tags: ["custom"] })
      return <>
        <output aria-label="form state">{JSON.stringify(object)}</output>
        <PolicyFormFields
          fields={[
            { path: "strategy", label: "defaultDomainResolverStrategy", kind: "select", options: ["prefer_ipv4", "prefer_ipv6"] },
            { path: "tags", label: "ruleSet", kind: "ref-multi", ref: "rule-set" },
          ]}
          object={object}
          namespace="policy.route"
          context={{ ruleSetTags: ["geo"] }}
          onChange={setObject}
        />
      </>
    }
    renderApp(<Harness />)
    expect(screen.getByLabelText("规则集 custom")).toBeInTheDocument()
    await user.click(screen.getByRole("combobox", { name: "默认域名解析策略" }))
    await user.click(await screen.findByRole("option", { name: "未设置" }))
    expect(screen.getByLabelText("form state")).not.toHaveTextContent("strategy")
  })
})

describe("policy tag helper edge cases", () => {
  it("policyConfigTags skips empty tags", () => {
    expect(policyConfigTags([{ tag: "" }, { tag: "ok" }, { tag: "ok" }])).toEqual(["ok"])
    expect(policyDNSServerTags({ servers: "bad" })).toEqual([])
    expect(policyRuleSetTags({ rule_set: [{ tag: 1 }, { tag: "geo" }] })).toEqual(["geo"])
    expect(groupPolicyFieldsBySection([])).toEqual([])
  })
})
