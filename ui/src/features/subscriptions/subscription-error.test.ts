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
    expect(subscriptionErrorHintKey("empty_content")).toBe("subscriptions.errorHintEmpty")
    expect(subscriptionErrorHintKey("content_too_large")).toBe("subscriptions.errorHintContentTooLarge")
    expect(subscriptionErrorHintKey("blocked_url")).toBe("subscriptions.errorHintBlockedURL")
    expect(subscriptionErrorHintKey("sync_failed")).toBe("subscriptions.errorHintSyncFailed")
    expect(subscriptionErrorHintKey("nope")).toBe("subscriptions.errorHintUnknown")
  })

  it("classifies common refresh failure messages", () => {
    expect(classifySubscriptionErrorMessage("subscription HTTP 403")).toBe("forbidden")
    expect(classifySubscriptionErrorMessage("subscription content produced no nodes")).toBe("empty_content")
    expect(classifySubscriptionErrorMessage("subscription content is too large")).toBe("content_too_large")
    expect(classifySubscriptionErrorMessage("configuration sync failed")).toBe("sync_failed")
    expect(classifySubscriptionErrorMessage("i/o timeout")).toBe("timeout")
    expect(classifySubscriptionErrorMessage("connection refused")).toBe("network")
    expect(classifySubscriptionErrorMessage("subscription URL targets a private or local address")).toBe("blocked_url")
  })

  it("prefers stored error_code", () => {
    expect(resolveSubscriptionErrorCode({ error: "x", error_code: "network" })).toBe("network")
    expect(resolveSubscriptionErrorCode({ error: "subscription HTTP 401" })).toBe("unauthorized")
    expect(resolveSubscriptionErrorCode({})).toBeUndefined()
  })

  it("classifies request-level failures and formats toast/clipboard", () => {
    expect(classifySubscriptionRequestError(new ApiError("missing", 404, "subscription_not_found"))).toBe("not_found")
    expect(classifySubscriptionRequestError(new ApiError("subscription URL targets a private or local address", 400, "invalid_request"))).toBe("blocked_url")
    expect(classifySubscriptionRequestError(new ApiError("subscription HTTP 403", 500, "subscription_refresh_failed"))).toBe("forbidden")
    expect(classifySubscriptionRequestError(new ApiError("sync failed", 500, "subscription_sync_failed"))).toBe("sync_failed")
    expect(formatSubscriptionRequestErrorToast(new ApiError("boom", 500, "subscription_refresh_failed"), "fallback")).toBe("boom")
    expect(subscriptionRequestErrorClipboardText(new Error("timeout"), { scope: "refresh", name: "cf" })).toContain("code: timeout")
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
})
