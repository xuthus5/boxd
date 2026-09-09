import { screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { RuleSetClientStatus } from "@/features/policy/ruleset-client-status"
import { RouteRuleSetCard } from "@/features/policy/route-rule-set-card"
import { classifyRuleSetErrorMessage, ruleSetErrorHintKey, summarizeRuleSetUpdate } from "@/features/policy/ruleset-update-error"
import { renderApp } from "@/test/render"

describe("rule set HTTP clients", () => {
  it("shows each expanded tag and describes kernel-managed downloads without exposing client options", () => {
    const base = { type: "remote", builtin: false, updatable: false, kernel_managed: true, note_code: "kernel_managed" }
    renderApp(<RouteRuleSetCard item={{ type: "remote", tag: ["cn", "private"], url: "https://example.org/{tag}.srs" }}
      status={{ ...base, tag: "cn" }} statuses={[
        { ...base, tag: "cn", http_client: "downloads", http_client_source: "rule_set" },
        { ...base, tag: "private", http_client_source: "inline" },
      ]} onEdit={vi.fn()} onCopy={vi.fn()} onDelete={vi.fn()} />)
    expect(screen.getByRole("button", { name: "编辑规则集 cn, private" })).toBeInTheDocument()
    expect(screen.getByText("HTTP 客户端：downloads（规则集指定）")).toBeInTheDocument()
    expect(screen.getByText("HTTP 客户端：内联或默认客户端（内联配置）")).toBeInTheDocument()
    expect(screen.getAllByText(/由内核按 update_interval/)).toHaveLength(2)
    expect(screen.queryByRole("button", { name: /更新规则集/ })).not.toBeInTheDocument()
    expect(screen.queryByText("kernel_managed")).not.toBeInTheDocument()
  })

  it("handles unknown sources and ordinary file notes", () => {
    renderApp(<RuleSetClientStatus statuses={[
      { tag: "one", type: "remote", builtin: false, updatable: false, http_client: "remote", http_client_source: "future", note_code: "kernel_managed" },
      { tag: "two", type: "local", builtin: false, updatable: true, note: "文件就绪" },
      { tag: "three", type: "local", builtin: false, updatable: false },
    ]} />)
    expect(screen.getByText("HTTP 客户端：remote（未指定来源）")).toBeInTheDocument()
    expect(screen.getByText("文件就绪")).toBeInTheDocument()
  })

  it("treats kernel-managed updates as skipped instead of failed", () => {
    expect(classifyRuleSetErrorMessage("kernel-managed rule-set update")).toBe("kernel_managed")
    expect(ruleSetErrorHintKey("kernel_managed")).toBe("kernel114.ruleSetKernelManaged")
    expect(summarizeRuleSetUpdate({ skipped_count: 1, failed_count: 0, results: [{ tag: "remote", type: "remote", ok: false, error_code: "kernel_managed", error: "kernel managed" }] }).failedSamples).toEqual([])
    renderApp(<RouteRuleSetCard item={{ type: "remote", tag: "remote" }}
      lastUpdate={{ tag: "remote", type: "remote", ok: false, error_code: "kernel_managed", error: "kernel managed" }}
      onEdit={vi.fn()} onCopy={vi.fn()} onDelete={vi.fn()} />)
    expect(screen.getByText(/由内核按 update_interval/)).toBeInTheDocument()
    expect(screen.queryByText("kernel_managed")).not.toBeInTheDocument()
  })
})
