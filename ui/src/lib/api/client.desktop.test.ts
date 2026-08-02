import { afterEach, describe, expect, it, vi } from "vitest"

import { apiRequest, apiRequestEnvelope } from "@/lib/api/client"
import { sessionStore } from "@/lib/session"

vi.mock("@/lib/api/desktop", () => ({
  isDesktop: vi.fn(),
  desktopRequest: vi.fn(),
}))

import { desktopRequest, isDesktop } from "@/lib/api/desktop"

afterEach(() => {
  sessionStore.clear()
  vi.clearAllMocks()
  vi.unstubAllGlobals()
})

describe("apiRequestEnvelope desktop mode", () => {
  it("routes GET requests through the bridge", async () => {
    vi.mocked(isDesktop).mockReturnValue(true)
    vi.mocked(desktopRequest).mockResolvedValue({ running: true })

    const result = await apiRequest<{ running: boolean }>("/api/service/status")
    expect(desktopRequest).toHaveBeenCalledWith("/api/service/status", "GET", undefined)
    expect(result).toEqual({ running: true })
  })

  it("routes POST requests with body through the bridge", async () => {
    vi.mocked(isDesktop).mockReturnValue(true)
    vi.mocked(desktopRequest).mockResolvedValue({ changed: true })

    const result = await apiRequest<{ changed: boolean }>("/api/settings/kernel-autostart", {
      method: "PUT",
      body: JSON.stringify({ enabled: true }),
    })
    expect(desktopRequest).toHaveBeenCalledWith("/api/settings/kernel-autostart", "PUT", JSON.stringify({ enabled: true }))
    expect(result).toEqual({ changed: true })
  })

  it("falls back to fetch when bridge reports unknown path", async () => {
    vi.mocked(isDesktop).mockReturnValue(true)
    vi.mocked(desktopRequest).mockRejectedValue(new Error("unknown path \"/api/unknown\""))
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ ok: true }), { status: 200 }),
    )
    vi.stubGlobal("fetch", fetchMock)

    const result = await apiRequest<{ ok: boolean }>("/api/unknown")
    expect(result).toEqual({ ok: true })
  })

  it("throws ApiError for non-path bridge errors", async () => {
    vi.mocked(isDesktop).mockReturnValue(true)
    vi.mocked(desktopRequest).mockRejectedValue(new Error("boom"))
    await expect(apiRequest("/api/config/")).rejects.toMatchObject({ status: 500 })
  })

  it("uses web fetch when not in desktop mode", async () => {
    vi.mocked(isDesktop).mockReturnValue(false)
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ status: "ok", data: { value: 1 }, error: null, meta: null }), { status: 200 }),
    )
    vi.stubGlobal("fetch", fetchMock)

    const result = await apiRequest<{ value: number }>("/api/config/")
    expect(result).toEqual({ value: 1 })
    expect(fetchMock).toHaveBeenCalled()
  })
})

describe("apiRequestEnvelope unwrap", () => {
  it("unwraps envelope data in desktop mode", async () => {
    vi.mocked(isDesktop).mockReturnValue(true)
    vi.mocked(desktopRequest).mockResolvedValue({ value: 2 })
    const result = await apiRequestEnvelope<{ value: number }>("/api/stats/traffic/history")
    expect(result.data).toEqual({ value: 2 })
  })
})
