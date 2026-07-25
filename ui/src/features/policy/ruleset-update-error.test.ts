import { describe, expect, it } from "vitest"

import {
  classifyRuleSetErrorMessage,
  classifyRuleSetRequestError,
  formatRuleSetFailureSample,
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
    expect(classifyRuleSetErrorMessage("subscription URL targets a private or local address")).toBe("blocked_url")
    expect(classifyRuleSetErrorMessage("empty rule-set body")).toBe("empty_content")
    expect(classifyRuleSetErrorMessage("rule-set content is too large")).toBe("content_too_large")
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

  it("covers every ruleset message classification", () => {
    const cases = [
      [undefined, "unknown"],
      ["", "unknown"],
      ["rule-set is not updatable", "not_updatable"],
      ["remote is not auto-updated", "unsupported"],
      ["format is not supported", "unsupported"],
      ["url is empty", "invalid_url"],
      ["invalid URL", "invalid_url"],
      ["unsupported protocol", "invalid_url"],
      ["private or local address", "blocked_url"],
      ["dial address is not public", "blocked_url"],
      ["content is too large", "content_too_large"],
      ["content too large", "content_too_large"],
      ["empty rule-set", "empty_content"],
      ["empty body", "empty_content"],
      ["empty content", "empty_content"],
      ["unexpected status 500", "http_status"],
      ["permission denied", "permission"],
      ["operation not permitted", "permission"],
      ["cache is unavailable", "cache"],
      ["bbolt transaction failed", "cache"],
      ["request deadline exceeded", "timeout"],
      ["i/o timeout", "timeout"],
      ["connection refused", "network"],
      ["connection reset", "network"],
      ["no such host", "network"],
      ["network unavailable", "network"],
      ["tls: handshake failure", "network"],
      ["x509: certificate", "network"],
      ["unexpected failure", "unknown"],
    ] as const

    for (const [message, expectedCode] of cases) {
      const actualCode = classifyRuleSetErrorMessage(message)
      expect(actualCode).toBe(expectedCode)
    }
    expect(ruleSetErrorHintKey()).toBe("policy.route.errorHintUnknown")
    expect(ruleSetErrorHintKey("future")).toBe("policy.route.errorHintUnknown")
  })

  it("handles empty results, defaults, and sample limits", () => {
    const expectedEmpty = {
      updated: 0,
      failed: 0,
      skipped: 0,
      restarted: false,
      failedSamples: [],
    }
    const actualEmpty = summarizeRuleSetUpdate()
    expect(actualEmpty).toEqual(expectedEmpty)
    expect(summarizeRuleSetUpdate(null)).toEqual(expectedEmpty)

    const actual = summarizeRuleSetUpdate({
      results: [
        { tag: "ok", type: "remote", ok: true },
        { tag: "inline", type: "inline", ok: false, error_code: "not_updatable" },
        { tag: "unsupported", type: "remote", ok: false, error_code: "unsupported" },
        { tag: " ", type: "remote", ok: false, error: " " },
        { tag: "unknown", type: "remote", ok: false, error: "unknown", error_code: "unknown" },
        { tag: "coded", type: "remote", ok: false, error: "network down", error_code: "network" },
        { tag: "ignored", type: "remote", ok: false, error: "fourth" },
      ],
    })
    const expectedSamples = [
      { tag: "—", error: "failed" },
      { tag: "unknown", error: "unknown" },
      { tag: "coded", error: "network down", code: "network" },
    ]
    expect(actual.failedSamples).toEqual(expectedSamples)
    expect(actual.updated).toBe(0)
    expect(actual.failed).toBe(0)
    expect(actual.skipped).toBe(0)
    expect(actual.restarted).toBe(false)
    expect(formatRuleSetFailureSample({ tag: "tag", error: "same", code: "same" })).toBe("tag: same")
    expect(formatRuleSetFailureSample({ tag: "tag", error: "error", code: "network" })).toBe("tag: network: error")
  })

  it("formats all update message and tone variants", () => {
    const zero = { updated: 0, failed: 0, skipped: 0, restarted: false, failedSamples: [] }
    const failed = { updated: 0, failed: 2, skipped: 0, restarted: false, failedSamples: [] }
    const successful = { updated: 2, failed: 0, skipped: 0, restarted: false, failedSamples: [] }
    const expectedZero = "ok 0"
    const actualZero = formatRuleSetUpdateMessage(zero, t)
    expect(actualZero).toBe(expectedZero)
    expect(formatRuleSetUpdateMessage(failed, t)).toContain("failed 2")
    expect(formatRuleSetUpdateMessage(successful, t)).toContain("ok 2")
    expect(formatRuleSetUpdateMessage({
      updated: 2,
      failed: 1,
      skipped: 3,
      restarted: true,
      failedSamples: [{ tag: "geo", error: "boom", code: "network" }],
    }, t)).toBe("partial 2/1 · skipped 3 · restarted · samples geo: network: boom")
    expect(ruleSetUpdateToastTone(zero)).toBe("success")
    expect(ruleSetUpdateToastTone(failed)).toBe("error")
    expect(ruleSetUpdateToastTone({ ...failed, updated: 1 })).toBe("warning")
  })

  it("formats optional clipboard fields and request failures", () => {
    const expectedEmpty = ""
    const actualSuccess = ruleSetUpdateErrorClipboardText({ tag: "geo", type: "remote", ok: true })
    expect(actualSuccess).toBe(expectedEmpty)
    expect(ruleSetUpdateErrorClipboardText({ tag: " ", type: " ", ok: false })).toBe("")
    expect(ruleSetUpdateErrorClipboardText({ tag: "geo", type: "remote", ok: false, error: "boom" }))
      .toBe("tag: geo\ntype: remote\ncode: unknown\nerror: boom")
    expect(ruleSetUpdateErrorClipboardText({ tag: "geo", type: "remote", ok: false, error_code: "network" }))
      .toBe("tag: geo\ntype: remote\ncode: network")

    const requestCases = [
      [{ code: "blocked_url" }, "blocked_url"],
      [{ code: "unavailable" }, "network"],
      [{ code: "timeout" }, "timeout"],
      [{ code: "future" }, "unknown"],
      [new Error("connection refused"), "network"],
      ["", "unknown"],
    ] as const
    for (const [error, expectedCode] of requestCases) {
      const actualCode = classifyRuleSetRequestError(error)
      expect(actualCode).toBe(expectedCode)
    }
    expect(ruleSetRequestErrorClipboardText("")).toBe("")
    expect(ruleSetRequestErrorClipboardText("boom", "probe")).toBe("scope: probe\ncode: unknown\nerror: boom")
    expect(formatRuleSetRequestErrorToast(new Error("  "), (key) => key, "fallback")).toBe("fallback")
    expect(formatRuleSetRequestErrorToast(new Error("connection refused"), (key) => key, "fallback"))
      .toBe("network: connection refused")
    expect(formatRuleSetRequestErrorToast("", (key) => key, "fallback")).toBe("fallback")
    expect(ruleSetBatchFailureClipboardText({
      updated: 0, failed: 0, skipped: 0, restarted: false, failedSamples: [],
    })).toBe("")
    expect(ruleSetBatchFailureClipboardText({
      updated: 0,
      failed: 1,
      skipped: 0,
      restarted: false,
      failedSamples: [{ tag: "geo", error: "boom" }],
    })).toBe("tag: geo\nerror: boom")
  })

})
