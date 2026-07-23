import { describe, expect, it } from "vitest"

import {
  shouldShowStreamStatus,
  streamStatusLabelKey,
  streamStatusVariant,
} from "@/features/observability/stream-status"

describe("streamStatus helpers", () => {
  it("prefers paused over transport status", () => {
    expect(streamStatusLabelKey("open", true)).toBe("observability.paused")
    expect(streamStatusVariant("open", true)).toBe("outline")
    expect(shouldShowStreamStatus("closed", true)).toBe(true)
  })

  it("maps live transport states", () => {
    expect(streamStatusLabelKey("connecting", false)).toBe("observability.streamConnecting")
    expect(streamStatusLabelKey("reconnecting", false)).toBe("observability.streamReconnecting")
    expect(streamStatusLabelKey("open", false)).toBe("observability.streamLive")
    expect(streamStatusLabelKey("closed", false)).toBe("")
    expect(streamStatusVariant("reconnecting", false)).toBe("destructive")
    expect(streamStatusVariant("open", false)).toBe("default")
    expect(shouldShowStreamStatus("closed", false)).toBe(false)
  })

  it("surfaces closed streams with an error", () => {
    expect(streamStatusLabelKey("closed", false, true)).toBe("observability.streamDisconnected")
    expect(streamStatusVariant("closed", false, true)).toBe("destructive")
    expect(shouldShowStreamStatus("closed", false, true)).toBe(true)
    expect(streamStatusLabelKey("closed", true, true)).toBe("observability.paused")
  })
})
