import { screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { DNSRuleDialog } from "@/features/policy/dns-rule-dialog"
import type { JsonObject } from "@/features/policy/policy-form-model"
import { renderApp } from "@/test/render"

const logicalCases: readonly [string, JsonObject, boolean][] = [
  ["missing children", { type: "logical", mode: "and", action: "reject" }, false],
  ["empty children", { type: "logical", mode: "and", rules: [], action: "reject" }, false],
  ["non-object child", { type: "logical", mode: "and", rules: [1], action: "reject" }, false],
  ["object child", { type: "logical", mode: "and", rules: [{}], action: "reject" }, true],
]

const predefinedCases: readonly [string, JsonObject][] = [
  ["empty", { action: "predefined" }],
  ["answer", { action: "predefined", answer: ["example. 60 IN A 192.0.2.1"] }],
  ["authority", { action: "predefined", ns: ["example. 60 IN NS ns.example."] }],
  ["extra", { action: "predefined", extra: ["ns.example. 60 IN A 192.0.2.53"] }],
]

describe("DNS logical rule requirements", () => {
  it.each(logicalCases)("handles %s", (_name, item, enabled) => {
    renderApp(<DNSRuleDialog open title="编辑 DNS 规则" item={item} serverTags={[]}
      onOpenChange={vi.fn()} onSave={vi.fn()} />)

    if (enabled) expect(screen.getByRole("button", { name: "保存" })).toBeEnabled()
    else expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
  })
})

describe("DNS predefined action requirements", () => {
  it.each(predefinedCases)("allows %s predefined response without rcode", (_name, item) => {
    renderApp(<DNSRuleDialog open title="编辑 DNS 规则" item={item} serverTags={[]}
      onOpenChange={vi.fn()} onSave={vi.fn()} />)

    expect(screen.getByRole("button", { name: "保存" })).toBeEnabled()
  })
})
