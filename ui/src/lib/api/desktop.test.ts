import { afterEach, describe, expect, it, vi } from "vitest"

import { autoLogin, callBridge, desktopGet, desktopRequest, isDesktop, streamEventName, subscribeDesktopStream } from "@/lib/api/desktop"

vi.mock("@/lib/api/bindings/github.com/xuthus5/boxd/desktop/boxdbridgeservice", () => ({
  Call: vi.fn(),
}))
vi.mock("@/lib/api/bindings/github.com/xuthus5/boxd/desktop/boxdauthservice", () => ({
  AutoLogin: vi.fn(),
}))

afterEach(() => {
  vi.unstubAllGlobals()
  vi.clearAllMocks()
})

// 文件级 mock：@wailsio/runtime 的 Events.On 捕获回调。
const runtimeListeners: Array<(ev: { data: unknown }) => void> = []
vi.mock("@wailsio/runtime", () => ({
  Events: {
    On: vi.fn((_name: string, cb: (ev: { data: unknown }) => void) => {
      runtimeListeners.push(cb)
      return () => {}
    }),
  },
}))

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
    expect(bridge.Call).toHaveBeenCalledWith({ path: "/api/service/status", method: "GET" })
    expect(resp).toEqual({ data: { running: true }, status: "ok" })
  })

  it("parses string body into an object for the wails bridge", async () => {
    const bridge = await import("@/lib/api/bindings/github.com/xuthus5/boxd/desktop/boxdbridgeservice")
    vi.mocked(bridge.Call).mockResolvedValue({ data: {}, status: "ok" })
    await callBridge("/api/config/", "PUT", '{"log":{"level":"info"}}')
    expect(bridge.Call).toHaveBeenCalledWith({
      path: "/api/config/",
      method: "PUT",
      body: { log: { level: "info" } },
    })
  })

  it("passes object body through unchanged", async () => {
    const bridge = await import("@/lib/api/bindings/github.com/xuthus5/boxd/desktop/boxdbridgeservice")
    vi.mocked(bridge.Call).mockResolvedValue({ data: {}, status: "ok" })
    await callBridge("/api/settings/kernel-autostart", "PUT", { enabled: true })
    expect(bridge.Call).toHaveBeenCalledWith({
      path: "/api/settings/kernel-autostart",
      method: "PUT",
      body: { enabled: true },
    })
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

describe("streamEventName", () => {
  it("maps known SSE paths to event names", () => {
    expect(streamEventName("/api/stats/traffic")).toBe("boxd:traffic")
    expect(streamEventName("/api/stats/connections")).toBe("boxd:connections")
    expect(streamEventName("/api/stats/logs")).toBe("boxd:kernel-log")
    expect(streamEventName("/api/stats/app-logs")).toBe("boxd:app-log")
  })

  it("returns null for unknown paths", () => {
    expect(streamEventName("/api/unknown")).toBeNull()
  })
})

describe("subscribeDesktopStream", () => {
  it("subscribes to the mapped event and forwards data", async () => {
    runtimeListeners.length = 0
    const onEvent = vi.fn()
    const onStatus = vi.fn()
    const stop = await subscribeDesktopStream<{ level: string }>("/api/stats/logs", { onEvent, onStatus })
    expect(runtimeListeners).toHaveLength(1)
    runtimeListeners[0]({ data: { level: "info", message: "hello" } })
    expect(onEvent).toHaveBeenCalledWith({ level: "info", message: "hello" })
    stop()
    expect(onStatus).toHaveBeenCalledWith("closed")
  })

  it("filters heartbeat events", async () => {
    runtimeListeners.length = 0
    const onEvent = vi.fn()
    await subscribeDesktopStream<{ type: string }>("/api/stats/traffic", { onEvent })
    runtimeListeners[0]({ data: { type: "heartbeat" } })
    expect(onEvent).not.toHaveBeenCalled()
  })

  it("closes immediately for unknown paths", async () => {
    const onStatus = vi.fn()
    const stop = await subscribeDesktopStream("/api/unknown", { onEvent: vi.fn(), onStatus })
    expect(onStatus).toHaveBeenCalledWith("closed")
    stop()
  })
})

describe("desktopRequest", () => {
  it("passes method and body to the bridge", async () => {
    const bridge = await import("@/lib/api/bindings/github.com/xuthus5/boxd/desktop/boxdbridgeservice")
    vi.mocked(bridge.Call).mockResolvedValue({ data: { ok: true }, status: "ok" })
    const data = await desktopRequest("/api/settings/kernel-autostart", "PUT", { enabled: true })
    expect(bridge.Call).toHaveBeenCalledWith({
      path: "/api/settings/kernel-autostart",
      method: "PUT",
      body: { enabled: true },
    })
    expect(data).toEqual({ ok: true })
  })

  it("throws when bridge reports an error", async () => {
    const bridge = await import("@/lib/api/bindings/github.com/xuthus5/boxd/desktop/boxdbridgeservice")
    vi.mocked(bridge.Call).mockResolvedValue({ error: "boom", status: "error" })
    await expect(desktopRequest("/api/service/start", "POST")).rejects.toThrow("boom")
  })
})
