import { describe, expect, it } from "vitest"

import {
  DEFAULT_SPEED_TEST_URL,
  isSpeedTestURLPreset,
  resolveInitialSpeedTestURL,
  resolveSpeedTestURLMode,
  SPEED_TEST_URL_PRESETS,
} from "@/lib/speed-test-urls"

describe("speed-test-urls", () => {
  it("prefers Cloudflare as the first default preset", () => {
    expect(SPEED_TEST_URL_PRESETS[0]).toBe("https://cp.cloudflare.com/")
    expect(DEFAULT_SPEED_TEST_URL).toBe("https://cp.cloudflare.com/")
  })

  it("classifies preset and custom URLs", () => {
    expect(isSpeedTestURLPreset("https://cp.cloudflare.com/")).toBe(true)
    expect(isSpeedTestURLPreset("https://example.com/test")).toBe(false)
    expect(resolveSpeedTestURLMode("")).toBe(DEFAULT_SPEED_TEST_URL)
    expect(resolveSpeedTestURLMode("https://captive.apple.com")).toBe("https://captive.apple.com")
    expect(resolveSpeedTestURLMode("https://example.com/test")).toBe("manual")
    expect(resolveInitialSpeedTestURL("")).toBe(DEFAULT_SPEED_TEST_URL)
    expect(resolveInitialSpeedTestURL("https://example.com")).toBe("https://example.com")
  })
})
