import { describe, expect, it } from "vitest"

import { ApiError } from "@/lib/api/client"
import {
  classifySubscriptionErrorMessage,
  classifySubscriptionRequestError,
  extractSubscriptionRefreshFailures,
  extractSubscriptionSyncError,
  formatSubscriptionRefreshBatchMessage,
  formatSubscriptionRequestErrorToast,
  resolveSubscriptionErrorCode,
  subscriptionErrorHintKey,
  subscriptionRefreshBatchClipboardText,
  subscriptionRequestErrorClipboardText,
  summarizeSubscriptionRefreshFailures,
} from "@/features/subscriptions/subscription-error"

describe("subscription error diagnostics", () => {
  it("maps codes to actionable hints", () => {
    expect(subscriptionErrorHintKey()).toBe("subscriptions.errorHintUnknown")
    expect(subscriptionErrorHintKey("empty_content")).toBe("subscriptions.errorHintEmpty")
    expect(subscriptionErrorHintKey("content_too_large")).toBe("subscriptions.errorHintContentTooLarge")
    expect(subscriptionErrorHintKey("blocked_url")).toBe("subscriptions.errorHintBlockedURL")
    expect(subscriptionErrorHintKey("sync_failed")).toBe("subscriptions.errorHintSyncFailed")
    expect(subscriptionErrorHintKey("nope")).toBe("subscriptions.errorHintUnknown")
  })

  it("classifies common refresh failure messages", () => {
    expect(classifySubscriptionErrorMessage()).toBe("unknown")
    expect(classifySubscriptionErrorMessage("subscription HTTP 401")).toBe("unauthorized")
    expect(classifySubscriptionErrorMessage("subscription HTTP 403")).toBe("forbidden")
    expect(classifySubscriptionErrorMessage("unauthorized response")).toBe("unauthorized")
    expect(classifySubscriptionErrorMessage("forbidden response")).toBe("forbidden")
    expect(classifySubscriptionErrorMessage("subscription content produced no nodes")).toBe("empty_content")
    expect(classifySubscriptionErrorMessage("subscription content is too large")).toBe("content_too_large")
    expect(classifySubscriptionErrorMessage("content too large")).toBe("content_too_large")
    expect(classifySubscriptionErrorMessage("configuration sync failed")).toBe("sync_failed")
    expect(classifySubscriptionErrorMessage("sync failed")).toBe("sync_failed")
    expect(classifySubscriptionErrorMessage("empty response")).toBe("empty_content")
    expect(classifySubscriptionErrorMessage("i/o timeout")).toBe("timeout")
    expect(classifySubscriptionErrorMessage("deadline exceeded")).toBe("timeout")
    expect(classifySubscriptionErrorMessage("connection refused")).toBe("network")
    expect(classifySubscriptionErrorMessage("no such host")).toBe("network")
    expect(classifySubscriptionErrorMessage("network unavailable")).toBe("network")
    expect(classifySubscriptionErrorMessage("subscription URL targets a private or local address")).toBe("blocked_url")
    expect(classifySubscriptionErrorMessage("private/local address blocked")).toBe("blocked_url")
    expect(classifySubscriptionErrorMessage("subscription HTTP 500")).toBe("http_status")
    expect(classifySubscriptionErrorMessage("unsupported protocol")).toBe("invalid_url")
    expect(classifySubscriptionErrorMessage("invalid URL")).toBe("invalid_url")
    expect(classifySubscriptionErrorMessage("://bad")).toBe("invalid_url")
    expect(classifySubscriptionErrorMessage("not found")).toBe("not_found")
    expect(classifySubscriptionErrorMessage("other")).toBe("unknown")
  })

  it("prefers stored error_code", () => {
    expect(resolveSubscriptionErrorCode({ error: "x", error_code: "network" })).toBe("network")
    expect(resolveSubscriptionErrorCode({ error: "subscription HTTP 401" })).toBe("unauthorized")
    expect(resolveSubscriptionErrorCode({ error_code: "stored" })).toBe("stored")
    expect(resolveSubscriptionErrorCode({})).toBeUndefined()
  })

  it("classifies request-level failures and formats toast/clipboard", () => {
    expect(classifySubscriptionRequestError(new ApiError("missing", 404, "subscription_not_found"))).toBe("not_found")
    expect(classifySubscriptionRequestError(new ApiError("subscription URL targets a private or local address", 400, "invalid_request"))).toBe("blocked_url")
    expect(classifySubscriptionRequestError(new ApiError("subscription HTTP 403", 500, "subscription_refresh_failed"))).toBe("forbidden")
    expect(classifySubscriptionRequestError(new ApiError("sync failed", 500, "subscription_sync_failed"))).toBe("sync_failed")
    expect(classifySubscriptionRequestError(new ApiError("mystery", 400, "invalid_request"))).toBe("invalid_url")
    expect(classifySubscriptionRequestError(new ApiError("missing", 404, "not_found"))).toBe("not_found")
    expect(classifySubscriptionRequestError(new ApiError("failed", 500, "subscription_refresh_failed"))).toBe("unknown")
    expect(classifySubscriptionRequestError(new ApiError("down", 503, "unavailable"))).toBe("network")
    expect(classifySubscriptionRequestError(new ApiError("slow", 408, "timeout"))).toBe("timeout")
    expect(classifySubscriptionRequestError(new ApiError("mystery", 418, ""))).toBe("unknown")
    expect(classifySubscriptionRequestError("subscription HTTP 401")).toBe("unauthorized")
    expect(formatSubscriptionRequestErrorToast(new ApiError("boom", 500, "subscription_refresh_failed"), "fallback")).toBe("boom")
    expect(subscriptionRequestErrorClipboardText(new Error("timeout"), { scope: "refresh", id: "  1 ", name: " cf " })).toContain("scope: refresh\nid: 1\nname: cf\ncode: timeout")
    expect(subscriptionRequestErrorClipboardText(new Error("timeout"), { scope: "refresh" })).toContain("scope: refresh")
    expect(subscriptionRequestErrorClipboardText(new Error("timeout"), { id: "1" })).toContain("id: 1")
    expect(subscriptionRequestErrorClipboardText(new Error("timeout"), { name: "cf" })).toContain("name: cf")
    expect(formatSubscriptionRequestErrorToast(null, "fallback")).toBe("fallback")
    expect(subscriptionRequestErrorClipboardText(null)).toBe("")
  })

  it("summarizes refresh-all failure samples", () => {
    const summary = summarizeSubscriptionRefreshFailures([
      { id: "1", name: "cf", code: "forbidden", message: "subscription HTTP 403" },
      { id: "2", name: "bad", code: "empty_content", message: "no nodes" },
    ])
    expect(summary.failed).toBe(2)
    expect(summary.failedSamples[0]).toMatchObject({ name: "cf", code: "forbidden" })
    const message = formatSubscriptionRefreshBatchMessage(summary, (key, values) => {
      if (key === "subscriptions.partialFailureCount") return `failed ${values?.count}`
      if (key === "subscriptions.refreshFailedSamples") return `samples ${values?.samples}`
      return key
    })
    expect(message).toContain("failed 2")
    expect(message).toContain("cf: forbidden")
    expect(subscriptionRefreshBatchClipboardText(summary)).toContain("name: cf")
    expect(extractSubscriptionRefreshFailures({ failed: [{ id: "1", name: "x", code: "network", message: "down" }] })).toEqual([
      { id: "1", name: "x", code: "network", message: "down" },
    ])
    expect(extractSubscriptionRefreshFailures(null)).toEqual([])
    expect(extractSubscriptionSyncError({ sync_error: " restart unavailable " })).toBe("restart unavailable")
    expect(extractSubscriptionSyncError({ sync_error: "   " })).toBeUndefined()
  })

  it("handles empty batch summaries and malformed refresh data", () => {
    expect(summarizeSubscriptionRefreshFailures(undefined)).toEqual({ failed: 0, failedSamples: [] })
    const summary = summarizeSubscriptionRefreshFailures([
      { id: "1", message: "unknown failure" },
      { id: "2" },
      { name: "third", code: "stored", message: "stored failure" },
      { name: "ignored fourth", message: "fourth" },
    ])
    expect(summary).toEqual({
      failed: 4,
      failedSamples: [
        { name: "1", message: "unknown failure" },
        { name: "2", message: "failed" },
        { name: "third", code: "stored", message: "stored failure" },
      ],
    })
    const translate = (key: string) => key
    expect(formatSubscriptionRefreshBatchMessage({ failed: 0, failedSamples: [] }, translate)).toBe("subscriptions.partialFailure")
    expect(formatSubscriptionRefreshBatchMessage({ failed: 1, failedSamples: [] }, translate)).toBe("subscriptions.partialFailureCount")
    expect(subscriptionRefreshBatchClipboardText({ failed: 0, failedSamples: [] })).toBe("")
    expect(subscriptionRefreshBatchClipboardText({ failed: 1, failedSamples: [{ name: "x", message: "bad" }] })).toBe("name: x\nerror: bad")
    expect(summarizeSubscriptionRefreshFailures([{ message: "unknown" }]).failedSamples[0]).toEqual({ name: "—", message: "unknown" })
    expect(extractSubscriptionRefreshFailures({ failed: [null, "bad", { id: 1, name: 2, code: false, message: null }] })).toEqual([
      { id: undefined, name: undefined, code: undefined, message: undefined },
    ])
    expect(extractSubscriptionRefreshFailures({ failed: "bad" })).toEqual([])
    expect(extractSubscriptionRefreshFailures("bad")).toEqual([])
    expect(extractSubscriptionSyncError(null)).toBeUndefined()
    expect(extractSubscriptionSyncError({ sync_error: 1 })).toBeUndefined()
  })
})
