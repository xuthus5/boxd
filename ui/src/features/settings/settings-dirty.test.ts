import { describe, expect, it } from "vitest"

import {
  isRuleSetAutoUpdateDirty,
  isRuleSetAutoUpdateReady,
  isTestURLDirty,
  isTestURLReady,
  isURLTestDefaultsDirty,
  isURLTestDefaultsReady,
} from "@/features/settings/settings-dirty"

const defaults = {
  enabled: true,
  url: "https://www.gstatic.com/generate_204",
  interval: "3m",
  tolerance: 50,
}

describe("settings dirty helpers", () => {
  it("gates test URL save on dirty valid http urls", () => {
    expect(isTestURLDirty("https://a.com", "https://a.com")).toBe(false)
    expect(isTestURLDirty("https://b.com", "https://a.com")).toBe(true)
    expect(isTestURLReady("https://a.com", "https://a.com")).toBe(false)
    expect(isTestURLReady("not-a-url", "https://a.com")).toBe(false)
    expect(isTestURLReady("https://b.com", "https://a.com")).toBe(true)
  })

  it("gates urltest defaults save on dirty valid fields", () => {
    expect(isURLTestDefaultsDirty(defaults, defaults)).toBe(false)
    expect(isURLTestDefaultsDirty({ ...defaults, interval: "5m" }, defaults)).toBe(true)
    expect(isURLTestDefaultsReady({
      ...defaults,
      interval: "5m",
      toleranceInput: "50",
    }, defaults)).toBe(true)
    expect(isURLTestDefaultsReady({
      ...defaults,
      interval: "bad",
      tolerance: 50,
      toleranceInput: "50",
    }, defaults)).toBe(false)
    expect(isURLTestDefaultsReady({
      ...defaults,
      toleranceInput: "50",
    }, defaults)).toBe(false)
  })

  it("gates ruleset auto-update save on dirty valid interval", () => {
    const saved = { enabled: false, interval: "24h" }
    expect(isRuleSetAutoUpdateDirty(saved, saved)).toBe(false)
    expect(isRuleSetAutoUpdateDirty({ enabled: true, interval: "24h" }, saved)).toBe(true)
    expect(isRuleSetAutoUpdateReady({ enabled: true, interval: "12h" }, saved, true)).toBe(true)
    expect(isRuleSetAutoUpdateReady({ enabled: true, interval: "bad" }, saved, false)).toBe(false)
  })
})
