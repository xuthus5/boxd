import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import {
  isNonEmptyJsonObjectArray,
  isPolicySectionStructureValid,
  type JsonObject,
} from "@/features/policy/policy-form-model"
import { PolicyPage, type PolicyVisualEditorProps } from "@/features/policy/policy-page"
import type { APIEnvelope, JsonValue, SingBoxConfig } from "@/lib/api/types"
import { renderApp } from "@/test/render"

const okEnvelope: APIEnvelope<JsonValue> = { status: "ok", data: null, error: null, meta: {} }

afterEach(() => vi.unstubAllGlobals())

describe("policy section structure", () => {
  it("requires configured section arrays to contain only objects", () => {
    expect(isPolicySectionStructureValid("route", {})).toBe(true)
    expect(isPolicySectionStructureValid("route", { rules: [{}], rule_set: [{ future: true }] })).toBe(true)
    expect(isPolicySectionStructureValid("dns", { servers: [{}], rules: [{}] })).toBe(true)
    for (const value of [null, "invalid", 1]) {
      expect(isPolicySectionStructureValid("route", { rules: [{}, value] } as JsonObject)).toBe(false)
      expect(isPolicySectionStructureValid("dns", { servers: [{}, value] } as JsonObject)).toBe(false)
    }
    expect(isPolicySectionStructureValid("route", { rule_set: {} })).toBe(false)
    expect(isPolicySectionStructureValid("dns", { rules: "invalid" })).toBe(false)
  })

  it("requires logical child arrays to be non-empty object arrays", () => {
    expect(isNonEmptyJsonObjectArray(undefined)).toBe(false)
    expect(isNonEmptyJsonObjectArray([])).toBe(false)
    expect(isNonEmptyJsonObjectArray([1])).toBe(false)
    expect(isNonEmptyJsonObjectArray([{}])).toBe(true)
  })
})

interface PageCase {
  section: "route" | "dns"
  invalid: JsonObject
  valid: JsonObject
}

const pageCases: PageCase[] = [
  {
    section: "route",
    invalid: { rules: [null, "invalid", 1], rule_set: [] },
    valid: { rules: [{ future_match: { enabled: true } }], rule_set: [{ type: "future", payload: 1 }] },
  },
  {
    section: "dns",
    invalid: { servers: {}, rules: [] },
    valid: { servers: [{ type: "future", payload: 1 }], rules: [{ future_match: { enabled: true } }] },
  },
]

function stubConfig(section: "route" | "dns", object: JsonObject) {
  const config: SingBoxConfig = { [section]: object }
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify(config))))
}

function renderPolicy(testCase: PageCase) {
  stubConfig(testCase.section, testCase.invalid)
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const renderVisual = ({ object }: PolicyVisualEditorProps) => (
    <button>可操作卡片 {JSON.stringify(object)}</button>
  )
  renderApp(<QueryClientProvider client={queryClient}><PolicyPage
    section={testCase.section} title="策略" installLabel="安装"
    install={() => Promise.resolve(okEnvelope)} renderVisual={renderVisual}
  /></QueryClientProvider>)
}

describe.each(pageCases)("$section page structure gate", (testCase) => {
  it("blocks visual operations and save until Advanced JSON fixes the arrays", async () => {
    const user = userEvent.setup()
    renderPolicy(testCase)

    expect(await screen.findByRole("alert")).toHaveTextContent("配置结构无效")
    expect(screen.queryByRole("button", { name: /可操作卡片/ })).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: "保存配置" })).toBeDisabled()

    await user.click(screen.getByRole("tab", { name: "高级 JSON" }))
    const editor = screen.getByRole("textbox", { name: "流量策略 JSON" })
    await user.click(editor)
    await user.keyboard("{Control>}a{/Control}")
    await user.paste(JSON.stringify(testCase.valid))
    await user.click(screen.getByRole("tab", { name: "可视化配置" }))

    expect(screen.queryByRole("alert")).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: /可操作卡片/ })).toHaveTextContent(JSON.stringify(testCase.valid))
    expect(screen.getByRole("button", { name: "保存配置" })).toBeEnabled()
  })
})
