import { afterEach, describe, expect, it } from "vitest"

import { preferencesStore } from "@/lib/storage"

afterEach(() => localStorage.clear())

describe("preferencesStore", () => {
  it("uses Chinese and system theme by default", () => {
    expect(preferencesStore.get()).toEqual({ language: "zh", theme: "system" })
  })

  it("persists supported preferences", () => {
    preferencesStore.set({ language: "en", theme: "dark" })
    expect(preferencesStore.get()).toEqual({ language: "en", theme: "dark" })
  })

  it("ignores malformed preferences", () => {
    localStorage.setItem("boxui.preferences.v1", "{}")
    expect(preferencesStore.get()).toEqual({ language: "zh", theme: "system" })
  })

  it("ignores unsupported preference values", () => {
    localStorage.setItem("boxui.preferences.v1", JSON.stringify({ language: "fr", theme: "blue" }))
    expect(preferencesStore.get()).toEqual({ language: "zh", theme: "system" })
  })
})
