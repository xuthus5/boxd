import { describe, expect, it } from "vitest"

import {
  classifySubscriptionErrorMessage,
  resolveSubscriptionErrorCode,
  subscriptionErrorHintKey,
} from "@/features/subscriptions/subscription-error"

describe("subscription error diagnostics", () => {
  it("maps known codes to hint keys", () => {
    expect(subscriptionErrorHintKey("empty_content")).toBe("subscriptions.errorHintEmpty")
    expect(subscriptionErrorHintKey("nope")).toBe("subscriptions.errorHintUnknown")
  })

  it("classifies free-form messages when code is missing", () => {
    expect(classifySubscriptionErrorMessage("subscription HTTP 403")).toBe("forbidden")
    expect(classifySubscriptionErrorMessage("subscription content produced no nodes")).toBe("empty_content")
    expect(classifySubscriptionErrorMessage("i/o timeout")).toBe("timeout")
  })

  it("prefers stored error_code", () => {
    expect(resolveSubscriptionErrorCode({ error: "x", error_code: "network" })).toBe("network")
    expect(resolveSubscriptionErrorCode({ error: "subscription HTTP 401" })).toBe("unauthorized")
    expect(resolveSubscriptionErrorCode({})).toBeUndefined()
  })
})
