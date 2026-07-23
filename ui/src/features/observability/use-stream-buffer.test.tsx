import { act, renderHook, waitFor } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { useStreamBuffer } from "@/features/observability/use-stream-buffer"

function controllableStream() {
  const encoder = new TextEncoder()
  let controller!: ReadableStreamDefaultController<Uint8Array>
  const stream = new ReadableStream<Uint8Array>({
    start(next) {
      controller = next
    },
  })
  return {
    stream,
    push(data: unknown) {
      controller.enqueue(encoder.encode(`data: ${JSON.stringify(data)}\n\n`))
    },
    close() {
      controller.close()
    },
  }
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.useRealTimers()
})

describe("useStreamBuffer", () => {
  it("tracks live events and marks the stream open", async () => {
    const body = controllableStream()
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(body.stream, { status: 200 })))

    const { result } = renderHook(() => useStreamBuffer<{ level: string }>("/api/stats/logs", "token", 10))
    await waitFor(() => expect(result.current.status).toBe("open"))

    act(() => {
      body.push({ level: "info" })
    })
    await waitFor(() => expect(result.current.items).toEqual([{ level: "info" }]))
    expect(result.current.error).toBe("")
    body.close()
  })

  it("exposes reconnecting status after a retryable failure", async () => {
    vi.useFakeTimers()
    const fetchMock = vi.fn().mockRejectedValue(new Error("offline"))
    vi.stubGlobal("fetch", fetchMock)

    const { result } = renderHook(() => useStreamBuffer("/api/stats/logs", "token"))
    await act(async () => {
      await Promise.resolve()
      await Promise.resolve()
    })
    expect(result.current.error).toContain("offline")
    expect(result.current.status).toBe("reconnecting")

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000)
      await Promise.resolve()
      await Promise.resolve()
    })
    expect(fetchMock.mock.calls.length).toBeGreaterThanOrEqual(2)
  })

  it("marks the stream closed while paused", async () => {
    const body = controllableStream()
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(body.stream, { status: 200 })))
    const { result } = renderHook(() => useStreamBuffer("/api/stats/logs", "token"))
    await waitFor(() => expect(result.current.status).toBe("open"))

    act(() => {
      result.current.setPaused(true)
    })
    await waitFor(() => expect(result.current.status).toBe("closed"))
    expect(result.current.paused).toBe(true)
    body.close()
  })
})
