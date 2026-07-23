import { describe, expect, it } from "vitest"

import {
  isOpenableSubscriptionURL,
  subscriptionErrorClipboardText,
  subscriptionSourceURL,
} from "@/features/subscriptions/subscription-error-actions"

describe("subscription error actions", () => {
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
})
