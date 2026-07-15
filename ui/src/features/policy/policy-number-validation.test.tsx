import { useState } from "react"
import { cleanup, fireEvent, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"

import { DNSGlobalCard } from "@/features/policy/dns-global-card"
import { DNSRuleDialog } from "@/features/policy/dns-rule-dialog"
import { DNSServerDialog } from "@/features/policy/dns-server-dialog"
import type { JsonObject } from "@/features/policy/policy-form-model"
import type { PolicyVisualEditorProps } from "@/features/policy/policy-page"
import { RouteGlobalCard } from "@/features/policy/route-global-card"
import { RouteRuleDialog } from "@/features/policy/route-rule-dialog"
import { renderApp } from "@/test/render"

function GlobalHarness({ kind }: { kind: "route" | "dns" }) {
  const [object, setObject] = useState<JsonObject>({})
  const [invalid, setInvalid] = useState(() => new Set<string>())
  const props: PolicyVisualEditorProps = {
    object,
    revision: 0,
    onChange: setObject,
    onFieldValidityChange: (path, valid) => setInvalid((current) => {
      const next = new Set(current)
      if (valid) next.delete(path)
      else next.add(path)
      return next
    }),
  }
  return <>{kind === "route" ? <RouteGlobalCard {...props} /> : <DNSGlobalCard {...props} />}
    <button disabled={invalid.size > 0}>保存测试</button></>
}

async function expectInvalidThenValid(label: string, invalid: string, valid: string, saveName = "保存") {
  const input = screen.getByLabelText(label)
  fireEvent.change(input, { target: { value: invalid } })
  expect(input).toHaveAttribute("aria-invalid", "true")
  expect(screen.getByRole("button", { name: saveName })).toBeDisabled()
  fireEvent.change(input, { target: { value: valid } })
  expect(input).toHaveAttribute("aria-invalid", "false")
  expect(screen.getByRole("button", { name: saveName })).toBeEnabled()
}

describe("global numeric validation", () => {
  it("gates Route and DNS global saves", async () => {
    renderApp(<GlobalHarness kind="route" />)
    await expectInvalidThenValid("默认标记", "-1", "0x10", "保存测试")
    cleanup()
    renderApp(<GlobalHarness kind="dns" />)
    await expectInvalidThenValid("缓存容量", "1.5", "1024", "保存测试")
  })
})

describe("dialog numeric validation", () => {
  it("gates Route matcher and action fields", async () => {
    renderApp(<RouteRuleDialog open title="编辑规则" item={{ action: "direct" }}
      onOpenChange={vi.fn()} onSave={vi.fn()} />)
    await userEvent.click(screen.getByRole("tab", { name: "端口与进程" }))
    await expectInvalidThenValid("源端口", "-1", "53")
    await userEvent.click(screen.getByRole("tab", { name: "执行动作" }))
    await expectInvalidThenValid("路由标记", "0x100000000", "0x10")
  })

  it("gates DNS server fields", async () => {
    renderApp(<DNSServerDialog open title="编辑 DNS 服务器"
      item={{ type: "udp", tag: "dns", server: "dns.example" }} onOpenChange={vi.fn()} onSave={vi.fn()} />)
    await expectInvalidThenValid("服务器端口", "65536", "53")
    await userEvent.click(screen.getByRole("tab", { name: "拨号与解析" }))
    await expectInvalidThenValid("解析重写 TTL", "-1", "60")
  })

  it("gates DNS rule matcher and action fields", async () => {
    renderApp(<DNSRuleDialog open title="编辑 DNS 规则" item={{ action: "reject" }} serverTags={[]}
      onOpenChange={vi.fn()} onSave={vi.fn()} />)
    await expectInvalidThenValid("IP 版本", "5", "4")
    await userEvent.click(screen.getByRole("tab", { name: "端口与环境" }))
    await expectInvalidThenValid("用户 ID", "-1", "1000")
  })
})
