import { describe, expect, it } from "vitest"

import {
  classifyRuleSetErrorMessage,
  classifyRuleSetRequestError,
  formatRuleSetRequestErrorToast,
  formatRuleSetUpdateMessage,
  resolveRuleSetErrorCode,
  ruleSetBatchFailureClipboardText,
  ruleSetErrorHintKey,
  ruleSetRequestErrorClipboardText,
  ruleSetUpdateErrorClipboardText,
  ruleSetUpdateToastTone,
  summarizeRuleSetUpdate,
} from "@/features/policy/ruleset-update-error"
import { ApiError } from "@/lib/api/client"

const t = (key: string, values?: Record<string, string | number>) => {
  if (key === "policy.route.ruleSetUpdateSuccess") return `ok ${values?.updated}`
  if (key === "policy.route.ruleSetUpdatePartial") return `partial ${values?.updated}/${values?.failed}`
  if (key === "policy.route.ruleSetUpdateFailed") return `failed ${values?.failed}`
  if (key === "policy.route.ruleSetUpdateSkipped") return `skipped ${values?.skipped}`
  if (key === "policy.route.ruleSetUpdateRestarted") return "restarted"
  if (key === "policy.route.ruleSetUpdateFailedSamples") return `samples ${values?.samples}`
  return key
}

describe("ruleset update error helpers", () => {
  it("classifies and resolves codes", () => {
    expect(classifyRuleSetErrorMessage("unexpected status 418")).toBe("http_status")
    expect(classifyRuleSetErrorMessage("remote rule-set url is empty")).toBe("invalid_url")
    expect(classifyRuleSetErrorMessage("empty rule-set body")).toBe("empty_content")
    expect(resolveRuleSetErrorCode({ error: "x", error_code: "network" })).toBe("network")
    expect(ruleSetErrorHintKey("timeout")).toBe("policy.route.errorHintTimeout")
  })

  it("summarizes batch update results with failed samples", () => {
    const summary = summarizeRuleSetUpdate({
      updated_count: 1,
      failed_count: 2,
      skipped_count: 1,
      restarted: true,
      results: [
        { tag: "geo", type: "remote", ok: true },
        { tag: "bad", type: "remote", ok: false, error: "unexpected status 500", error_code: "http_status" },
        { tag: "net", type: "remote", ok: false, error: "connection refused", error_code: "network" },
        { tag: "inline", type: "inline", ok: false, error: "rule-set is not updatable", error_code: "not_updatable" },
      ],
    })
    expect(summary).toEqual({
      updated: 1,
      failed: 2,
      skipped: 1,
      restarted: true,
      failedSamples: [
        { tag: "bad", error: "unexpected status 500", code: "http_status" },
        { tag: "net", error: "connection refused", code: "network" },
      ],
    })
    const message = formatRuleSetUpdateMessage(summary, t)
    expect(message).toContain("partial 1/2")
    expect(message).toContain("skipped 1")
    expect(message).toContain("restarted")
    expect(message).toContain("bad: http_status: unexpected status 500")
    expect(ruleSetUpdateToastTone(summary)).toBe("warning")
    expect(ruleSetUpdateToastTone({ updated: 0, failed: 1, skipped: 0, restarted: false, failedSamples: [] })).toBe("error")
  })

  it("formats clipboard diagnostics", () => {
    expect(ruleSetUpdateErrorClipboardText({
      tag: "geo",
      type: "remote",
      ok: false,
      error: "empty rule-set body",
      error_code: "empty_content",
    })).toBe([
      "tag: geo",
      "type: remote",
      "code: empty_content",
      "error: empty rule-set body",
    ].join("\n"))
    expect(ruleSetUpdateErrorClipboardText({ tag: "geo", type: "remote", ok: true })).toBe("")
  })

  it("classifies request-level update failures and batch clipboard", () => {
    expect(classifyRuleSetRequestError(new ApiError("timeout", 504, "timeout"))).toBe("timeout")
    expect(classifyRuleSetRequestError(new Error("connection refused"))).toBe("network")
    expect(ruleSetRequestErrorClipboardText(new Error("boom"))).toContain("code: unknown")
    expect(formatRuleSetRequestErrorToast(
      new ApiError("service not available", 503, "unavailable"),
      (k) => k,
      "fallback",
    )).toBe("network: service not available")
    const summary = summarizeRuleSetUpdate({
      updated_count: 0,
      failed_count: 1,
      skipped_count: 0,
      restarted: false,
      results: [{ tag: "geo", type: "remote", ok: false, error: "unexpected status 500", error_code: "http_status" }],
    })
    expect(ruleSetBatchFailureClipboardText(summary)).toContain("tag: geo")
    expect(ruleSetBatchFailureClipboardText(summary)).toContain("code: http_status")
  })

})
