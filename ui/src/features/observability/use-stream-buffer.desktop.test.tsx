import { act, renderHook, waitFor } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { useStreamBuffer } from "@/features/observability/use-stream-buffer"
import { sessionStore } from "@/lib/session"

vi.mock("@/lib/api/desktop", () => ({
  isDesktop: vi.fn(),
  subscribeDesktopStream: vi.fn(),
  desktopGet: vi.fn(),
}))

import { desktopGet, isDesktop, subscribeDesktopStream } from "@/lib/api/desktop"

afterEach(() => {
  sessionStore.clear()
  vi.clearAllMocks()
})

describe("useStreamBuffer desktop mode", () => {
  it("uses event subscription instead of SSE in desktop mode", async () => {
    vi.mocked(isDesktop).mockReturnValue(true)
    vi.mocked(desktopGet).mockResolvedValue({ entries: [] })
    vi.mocked(subscribeDesktopStream).mockResolvedValue(() => {})
    const { result } = renderHook(() => useStreamBuffer<{ level: string }>("/api/stats/logs", "token", 10))
    await waitFor(() => {
      expect(subscribeDesktopStream).toHaveBeenCalledWith(
        "/api/stats/logs",
        expect.objectContaining({ onEvent: expect.any(Function) }),
      )
    })
    expect(result.current.items).toEqual([])
  })

  it("seeds items from history snapshot before subscribing", async () => {
    let onEvent: ((event: { level: string }) => void) | undefined
    vi.mocked(isDesktop).mockReturnValue(true)
    vi.mocked(desktopGet).mockResolvedValue({ entries: [{ level: "info", message: "old" }] })
    vi.mocked(subscribeDesktopStream).mockImplementation(async (_path, options) => {
      onEvent = options.onEvent
      return () => {}
    })
    const { result } = renderHook(() => useStreamBuffer<{ level: string }>("/api/stats/logs", "token", 10))
    await waitFor(() => expect(result.current.items).toEqual([{ level: "info", message: "old" }]))
    expect(desktopGet).toHaveBeenCalledWith("/api/stats/logs")
    await waitFor(() => expect(onEvent).toBeDefined())
    act(() => {
      onEvent?.({ level: "info", message: "new" })
    })
    expect(result.current.items).toEqual([
      { level: "info", message: "old" },
      { level: "info", message: "new" },
    ])
  })

  it("subscribes even when history fetch fails", async () => {
    vi.mocked(isDesktop).mockReturnValue(true)
    vi.mocked(desktopGet).mockRejectedValue(new Error("bridge down"))
    vi.mocked(subscribeDesktopStream).mockResolvedValue(() => {})
    const { result } = renderHook(() => useStreamBuffer<{ level: string }>("/api/stats/logs", "token", 10))
    await waitFor(() => {
      expect(subscribeDesktopStream).toHaveBeenCalledWith(
        "/api/stats/logs",
        expect.objectContaining({ onEvent: expect.any(Function) }),
      )
    })
    expect(result.current.items).toEqual([])
  })

  it("appends received events", async () => {
    let onEvent: ((event: { level: string }) => void) | undefined
    vi.mocked(isDesktop).mockReturnValue(true)
    vi.mocked(desktopGet).mockResolvedValue({ entries: [] })
    vi.mocked(subscribeDesktopStream).mockImplementation(async (_path, options) => {
      onEvent = options.onEvent
      return () => {}
    })
    const { result } = renderHook(() => useStreamBuffer<{ level: string }>("/api/stats/logs", "token", 10))
    await waitFor(() => expect(onEvent).toBeDefined())
    act(() => {
      onEvent?.({ level: "info", message: "hello" })
    })
    expect(result.current.items).toEqual([{ level: "info", message: "hello" }])
  })

  it("does not re-subscribe while paused", async () => {
    vi.mocked(isDesktop).mockReturnValue(true)
    vi.mocked(desktopGet).mockResolvedValue({ entries: [] })
    vi.mocked(subscribeDesktopStream).mockResolvedValue(() => {})
    const { result } = renderHook(() => useStreamBuffer<{ level: string }>("/api/stats/logs", "token", 10))
    await waitFor(() => expect(subscribeDesktopStream).toHaveBeenCalledTimes(1))
    const callsBeforePause = vi.mocked(subscribeDesktopStream).mock.calls.length
    act(() => {
      result.current.setPaused(true)
      result.current.reconnect()
    })
    await waitFor(() => expect(result.current.paused).toBe(true))
    expect(vi.mocked(subscribeDesktopStream).mock.calls.length).toBe(callsBeforePause)
  })

  it("updates status when event subscription opens", async () => {
    let onStatus: ((status: string) => void) | undefined
    vi.mocked(isDesktop).mockReturnValue(true)
    vi.mocked(desktopGet).mockResolvedValue({ entries: [] })
    vi.mocked(subscribeDesktopStream).mockImplementation(async (_path, options) => {
      onStatus = options.onStatus
      return () => {}
    })
    const { result } = renderHook(() => useStreamBuffer<{ level: string }>("/api/stats/logs", "token", 10))
    await waitFor(() => expect(onStatus).toBeDefined())
    act(() => {
      onStatus?.("open")
    })
    expect(result.current.status).toBe("open")
  })
})
