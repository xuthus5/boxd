import { describe, expect, it, vi } from "vitest"

import { consumeSSEStream, openSSE } from "@/lib/api/sse"
import { setUnauthorizedHandler } from "@/lib/api/client"

function streamChunks(chunks: string[]) {
  const encoder = new TextEncoder()
  return new ReadableStream<Uint8Array>({
    start(controller) {
      chunks.forEach((chunk) => controller.enqueue(encoder.encode(chunk)))
      controller.close()
    },
  })
}

describe("consumeSSEStream", () => {
  it("parses events split across chunks", async () => {
    const onEvent = vi.fn()
    await consumeSSEStream(streamChunks([
      "data: {\"level\":\"info\",",
      "\"message\":\"ready\"}\n\n",
      ": keepalive\n\ndata: {\"level\":\"error\",\"message\":\"failed\"}\n\n",
    ]), onEvent)

    expect(onEvent).toHaveBeenNthCalledWith(1, { level: "info", message: "ready" })
    expect(onEvent).toHaveBeenNthCalledWith(2, { level: "error", message: "failed" })
  })

  it("rejects malformed event JSON", async () => {
    await expect(consumeSSEStream(streamChunks(["data: nope\n\n"]), vi.fn()))
      .rejects.toThrow("Invalid SSE event data")
  })

  it("opens an authenticated stream and returns a cleanup function", async () => {
    const onEvent = vi.fn()
    const fetchMock = vi.fn().mockResolvedValue(new Response(
      streamChunks(["data: {\"running\":true}\n\n"]),
      { status: 200 },
    ))
    vi.stubGlobal("fetch", fetchMock)

    const cleanup = openSSE({ path: "/api/stats/traffic", token: "token", onEvent })
    await vi.waitFor(() => expect(onEvent).toHaveBeenCalledWith({ running: true }))
    cleanup()

    const headers = new Headers(fetchMock.mock.calls[0][1]?.headers)
    expect(headers.get("Authorization")).toBe("Bearer token")
  })

  it("reports failed stream responses", async () => {
    const onError = vi.fn()
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(null, { status: 503 })))
    openSSE({ path: "/api/stats/logs", token: "token", onEvent: vi.fn(), onError })
    await vi.waitFor(() => expect(onError).toHaveBeenCalledOnce())
  })
})

describe("openSSE lifecycle", () => {
  it("stops reconnecting and reports unauthorized SSE responses", async () => {
    vi.useFakeTimers()
    const unauthorized = vi.fn()
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 401 }))
    setUnauthorizedHandler(unauthorized)
    vi.stubGlobal("fetch", fetchMock)
    openSSE({ path: "/api/stats/logs", token: "expired", onEvent: vi.fn() })
    await vi.advanceTimersByTimeAsync(10000)
    expect(unauthorized).toHaveBeenCalledOnce()
    expect(fetchMock).toHaveBeenCalledOnce()
    setUnauthorizedHandler(undefined)
    vi.useRealTimers()
  })

  it("ignores comment-only events and supports external cancellation", async () => {
    const onEvent = vi.fn()
    await consumeSSEStream(streamChunks([": keepalive\n\n"]), onEvent)
    expect(onEvent).not.toHaveBeenCalled()

    const controller = new AbortController()
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue("offline"))
    const cleanup = openSSE({ path: "/api/stats/logs", token: "token", signal: controller.signal, onEvent })
    controller.abort()
    cleanup()
  })
})

describe("openSSE retries", () => {
  it("normalizes non-error connection failures", async () => {
    const onError = vi.fn()
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue("offline"))
    openSSE({ path: "/api/stats/logs", token: "token", onEvent: vi.fn(), onError })
    await vi.waitFor(() => expect(onError).toHaveBeenCalledWith(expect.objectContaining({
      message: "SSE connection failed",
    })))
  })

  it("reconnects after a retryable connection failure", async () => {
    vi.useFakeTimers()
    const onEvent = vi.fn()
    const fetchMock = vi.fn()
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce(new Response(streamChunks(["data: {\"running\":true}\n\n"])))
    vi.stubGlobal("fetch", fetchMock)

    const cleanup = openSSE({ path: "/api/stats/traffic", token: "token", onEvent })
    await vi.advanceTimersByTimeAsync(1000)

    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(onEvent).toHaveBeenCalledWith({ running: true })
    cleanup()
    vi.useRealTimers()
  })

  it("cancels a scheduled reconnect during cleanup", async () => {
    vi.useFakeTimers()
    const fetchMock = vi.fn().mockRejectedValue(new Error("offline"))
    vi.stubGlobal("fetch", fetchMock)
    const cleanup = openSSE({ path: "/api/stats/logs", token: "token", onEvent: vi.fn() })
    await vi.advanceTimersByTimeAsync(0)
    cleanup()
    await vi.advanceTimersByTimeAsync(1000)
    expect(fetchMock).toHaveBeenCalledOnce()
    vi.useRealTimers()
  })
})
