import { afterEach, describe, expect, it, vi } from "vitest"

import { autoLogin, callBridge, desktopGet, isDesktop } from "@/lib/api/desktop"

vi.mock("@/lib/api/bindings/github.com/xuthus5/boxd/desktop/boxdbridgeservice", () => ({
  Call: vi.fn(),
}))
vi.mock("@/lib/api/bindings/github.com/xuthus5/boxd/desktop/boxdauthservice", () => ({
  AutoLogin: vi.fn(),
}))

afterEach(() => {
  vi.unstubAllGlobals()
})

describe("isDesktop", () => {
  it("returns false in web browser environment", () => {
    expect(isDesktop()).toBe(false)
  })

  it("returns true when wails runtime is injected", () => {
    ;(window as { _wails?: unknown })._wails = { runtime: {} }
    expect(isDesktop()).toBe(true)
    delete (window as { _wails?: unknown })._wails
  })
})

describe("callBridge", () => {
  it("calls the bridge binding with the path", async () => {
    const bridge = await import("@/lib/api/bindings/github.com/xuthus5/boxd/desktop/boxdbridgeservice")
    vi.mocked(bridge.Call).mockResolvedValue({ data: { running: true }, status: "ok" })
    const resp = await callBridge("/api/service/status")
    expect(bridge.Call).toHaveBeenCalledWith({ path: "/api/service/status" })
    expect(resp).toEqual({ data: { running: true }, status: "ok" })
  })
})

describe("desktopGet", () => {
  it("returns data on success", async () => {
    const bridge = await import("@/lib/api/bindings/github.com/xuthus5/boxd/desktop/boxdbridgeservice")
    vi.mocked(bridge.Call).mockResolvedValue({ data: { value: 1 }, status: "ok" })
    await expect(desktopGet("/api/stats/traffic/history")).resolves.toEqual({ value: 1 })
  })

  it("throws when bridge reports an error", async () => {
    const bridge = await import("@/lib/api/bindings/github.com/xuthus5/boxd/desktop/boxdbridgeservice")
    vi.mocked(bridge.Call).mockResolvedValue({ error: "boom", status: "error" })
    await expect(desktopGet("/api/unknown")).rejects.toThrow("boom")
  })
})

describe("autoLogin", () => {
  it("returns the embedded session", async () => {
    const bridge = await import("@/lib/api/bindings/github.com/xuthus5/boxd/desktop/boxdauthservice")
    vi.mocked(bridge.AutoLogin).mockResolvedValue({ token: "tok", expires_at: "2026-01-01T00:00:00Z" })
    const session = await autoLogin()
    expect(session.token).toBe("tok")
    expect(session.expires_at).toBe("2026-01-01T00:00:00Z")
  })
})
