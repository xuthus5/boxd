import { beforeEach, describe, expect, it, vi } from "vitest"

import { ApiError } from "@/lib/api/client"

vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }))

import { toast } from "sonner"

import {
  isOpenableSubscriptionURL,
  reportSubscriptionRefreshBatch,
  reportSubscriptionRequestError,
  subscriptionErrorClipboardText,
  subscriptionSourceURL,
} from "@/features/subscriptions/subscription-error-actions"

describe("subscription error actions", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("formats a diagnostic clipboard payload", () => {
    expect(subscriptionErrorClipboardText({
      name: "主订阅",
      url: "https://example.com/sub",
      error: "subscription HTTP 403",
      error_code: "forbidden",
      error_at: "2026-07-24T00:00:00Z",
    })).toBe([
      "name: 主订阅",
      "url: https://example.com/sub",
      "code: forbidden",
      "error: subscription HTTP 403",
      "at: 2026-07-24T00:00:00Z",
    ].join("\n"))
  })

  it("only allows http(s) open targets", () => {
    expect(isOpenableSubscriptionURL("https://example.com/a")).toBe(true)
    expect(isOpenableSubscriptionURL("http://example.com/a")).toBe(true)
    expect(isOpenableSubscriptionURL("ftp://example.com/a")).toBe(false)
    expect(isOpenableSubscriptionURL("not a url")).toBe(false)
    expect(subscriptionSourceURL("https://example.com/a")).toBe("https://example.com/a")
    expect(subscriptionSourceURL("bad")).toBe("")
  })

  it("reports request and batch failures with clipboard actions", () => {
    const t = (key: string) => key
    reportSubscriptionRequestError(new ApiError("subscription HTTP 403", 500, "subscription_refresh_failed"), t, {
      scope: "refresh",
      name: "cf",
    })
    expect(toast.error).toHaveBeenCalled()
    const [message, options] = vi.mocked(toast.error).mock.calls.at(-1)!
    expect(String(message)).toContain("forbidden")
    expect(options).toEqual(expect.objectContaining({
      description: "subscriptions.errorHintForbidden",
      action: expect.objectContaining({ label: "subscriptions.copyError" }),
    }))
    reportSubscriptionRefreshBatch(
      { failed: 1, failedSamples: [{ name: "cf", code: "forbidden", message: "subscription HTTP 403" }] },
      "batch failed",
      t,
    )
    expect(vi.mocked(toast.error).mock.calls.at(-1)?.[0]).toBe("batch failed")
  })
})
