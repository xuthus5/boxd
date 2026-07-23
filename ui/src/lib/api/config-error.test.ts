import { describe, expect, it } from "vitest"

import { extractConfigPath, formatConfigErrorMessage, parseConfigError } from "@/lib/api/config-error"

describe("config-error", () => {
  it("extracts dotted and indexed paths", () => {
    expect(extractConfigPath("inbounds[0].listen_port: missing")).toBe("inbounds[0].listen_port")
    expect(extractConfigPath("decode config at outbounds.0.server: invalid")).toBe("outbounds.0.server")
    expect(extractConfigPath("field route.rules[1].outbound: required")).toBe("route.rules[1].outbound")
  })

  it("builds a path-first summary", () => {
    const parsed = parseConfigError("inbounds[0].type: unknown type foo")
    expect(parsed.path).toBe("inbounds[0].type")
    expect(parsed.summary).toContain("inbounds[0].type")
    expect(formatConfigErrorMessage("plain failure")).toBe("plain failure")
  })

  it("handles empty messages", () => {
    expect(parseConfigError("").summary).toBe("invalid sing-box config")
  })
})
