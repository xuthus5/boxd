import { describe, expect, it } from "vitest"

import { rolledBackMessage, saveErrorMessage } from "@/lib/api/status"

describe("status helpers", () => {
  it("returns fallback when error is missing", () => {
    expect(rolledBackMessage({ error: null }, "rolled back")).toBe("rolled back")
  })

  it("formats restart failure detail", () => {
    expect(rolledBackMessage({
      error: { code: "config_restart_failed", message: "restart failed after config save: bind" },
    }, "rolled back")).toContain("bind")
  })

  it("formats save errors with config path when present", () => {
    expect(saveErrorMessage(new Error("inbounds[0].listen_port: invalid"))).toContain("inbounds[0].listen_port")
    expect(saveErrorMessage(new Error(""))).toBe("request failed")
  })
})
