import { afterEach, describe, expect, it } from "vitest"

import { sessionStore } from "@/lib/session"

afterEach(() => sessionStorage.clear())

describe("sessionStore", () => {
  it("persists and restores a valid session", () => {
    const session = { token: "token", expiresAt: "2099-01-01T00:00:00Z" }
    sessionStore.set(session)
    expect(sessionStore.get()).toEqual(session)
    expect(sessionStore.isValid()).toBe(true)
  })

  it("clears an expired session", () => {
    sessionStore.set({ token: "token", expiresAt: "2000-01-01T00:00:00Z" })
    expect(sessionStore.get()).toBeNull()
    expect(sessionStorage.length).toBe(0)
  })

  it("ignores malformed storage", () => {
    sessionStorage.setItem("boxui.session.v1", "not-json")
    expect(sessionStore.get()).toBeNull()
  })

  it("ignores values with the wrong shape", () => {
    sessionStorage.setItem("boxui.session.v1", JSON.stringify({ token: 1, expiresAt: null }))
    expect(sessionStore.get()).toBeNull()
    expect(sessionStore.isValid()).toBe(false)
  })
})
