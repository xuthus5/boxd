import { afterEach, describe, expect, it, vi } from "vitest"

import { ApiError, apiRequest, setUnauthorizedHandler } from "@/lib/api/client"
import { sessionStore } from "@/lib/session"

afterEach(() => {
  vi.unstubAllGlobals()
  sessionStore.clear()
  setUnauthorizedHandler(undefined)
})

describe("apiRequest", () => {
  it("returns direct JSON responses", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ running: true }), { status: 200 }),
    ))

    await expect(apiRequest<{ running: boolean }>("/api/service/status"))
      .resolves.toEqual({ running: true })
  })

  it("unwraps API response data", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ status: "ok", data: { changed: true }, error: null, meta: null })),
    ))

    await expect(apiRequest<{ changed: boolean }>("/api/settings/password"))
      .resolves.toEqual({ changed: true })
  })

  it("attaches the bearer token and JSON content type", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const fetchMock = vi.fn().mockResolvedValue(new Response("{}"))
    vi.stubGlobal("fetch", fetchMock)

    await apiRequest("/api/config", { method: "PUT", body: "{}" })

    const request = new Request("http://localhost/api/config", fetchMock.mock.calls[0][1])
    expect(request.headers.get("Authorization")).toBe("Bearer token")
    expect(request.headers.get("Content-Type")).toBe("application/json")
  })
})

describe("apiRequest error handling", () => {
  it("throws a typed error and reports unauthorized responses", async () => {
    const unauthorized = vi.fn()
    setUnauthorizedHandler(unauthorized)
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
      status: "error",
      data: null,
      error: { code: "unauthorized", message: "invalid token" },
      meta: null,
    }), { status: 401 })))

    const error = await apiRequest("/api/config").catch((reason: unknown) => reason)

    expect(error).toBeInstanceOf(ApiError)
    expect(error).toMatchObject({ status: 401, code: "unauthorized", message: "invalid token" })
    expect(unauthorized).toHaveBeenCalledOnce()
  })

  it("handles empty success responses", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(null, { status: 204 })))
    await expect(apiRequest<void>("/api/auth/logout")).resolves.toBeUndefined()
  })

  it("rejects invalid JSON and direct error bodies", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response("nope")))
    await expect(apiRequest("/api/config")).rejects.toMatchObject({ code: "invalid_response" })

    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response(JSON.stringify({
      code: "bad_gateway", message: "upstream failed",
    }), { status: 502 })))
    await expect(apiRequest("/api/config")).rejects.toMatchObject({
      status: 502, code: "bad_gateway", message: "upstream failed",
    })
  })

  it("uses status text for unstructured errors and preserves custom headers", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response("", { status: 500, statusText: "Broken" }))
    vi.stubGlobal("fetch", fetchMock)
    await expect(apiRequest("/api/test", { body: "x", headers: { "Content-Type": "text/plain" } }))
      .rejects.toMatchObject({ message: "Broken", code: "request_failed" })
    const headers = new Headers(fetchMock.mock.calls[0][1]?.headers)
    expect(headers.get("Content-Type")).toBe("text/plain")
  })
})
