import { describe, expect, it } from "vitest"

import {
  classifyStreamErrorMessage,
  formatStreamErrorTitle,
  streamErrorClipboardText,
  streamErrorHintKey,
} from "@/features/observability/stream-error"

describe("stream error helpers", () => {
  it("classifies common SSE failures", () => {
    expect(classifyStreamErrorMessage("SSE request failed with status 401")).toBe("unauthorized")
    expect(classifyStreamErrorMessage("SSE request failed with status 503")).toBe("unavailable")
    expect(classifyStreamErrorMessage("SSE request failed with status 404")).toBe("unavailable")
    expect(classifyStreamErrorMessage("Invalid SSE event data")).toBe("invalid_payload")
    expect(classifyStreamErrorMessage("request timeout")).toBe("timeout")
    expect(classifyStreamErrorMessage("Failed to fetch")).toBe("network")
    expect(classifyStreamErrorMessage("offline")).toBe("network")
    expect(classifyStreamErrorMessage("SSE connection failed")).toBe("network")
    expect(classifyStreamErrorMessage("weird")).toBe("unknown")
    expect(classifyStreamErrorMessage("")).toBe("unknown")
  })

  it("maps hint keys and clipboard diagnostics", () => {
    expect(streamErrorHintKey("unauthorized")).toBe("observability.errorHintStreamUnauthorized")
    expect(streamErrorHintKey("unavailable")).toBe("observability.errorHintStreamUnavailable")
    expect(streamErrorHintKey("network")).toBe("observability.errorHintStreamNetwork")
    expect(streamErrorHintKey("invalid_payload")).toBe("observability.errorHintStreamInvalidPayload")
    expect(streamErrorHintKey("timeout")).toBe("observability.errorHintStreamTimeout")
    expect(streamErrorHintKey("unknown")).toBe("observability.errorHintStreamUnknown")
    expect(streamErrorClipboardText({
      path: "/api/stats/logs",
      status: "reconnecting",
      paused: false,
      error: "SSE request failed with status 503",
    })).toBe([
      "path: /api/stats/logs",
      "status: reconnecting",
      "paused: false",
      "code: unavailable",
      "error: SSE request failed with status 503",
    ].join("\n"))
    expect(streamErrorClipboardText({ error: "   " })).toBe("")
    expect(formatStreamErrorTitle((key, values) => {
      if (key === "observability.streamErrorWithCode") return `err:${values?.code}`
      return "stream error"
    }, "network")).toBe("err:network")
    expect(formatStreamErrorTitle((key) => key, "unknown")).toBe("observability.streamError")
  })
})
