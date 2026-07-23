import { describe, expect, it } from "vitest"

import { rolledBackMessage } from "@/lib/api/status"

describe("rolledBackMessage", () => {
  it("returns fallback when error is missing", () => {
    expect(rolledBackMessage({ error: null }, "rolled back")).toBe("rolled back")
  })

  it("appends restart detail", () => {
    expect(rolledBackMessage({
      error: { code: "config_restart_failed", message: "restart failed after config save: bind" },
    }, "配置已回滚")).toBe("配置已回滚: restart failed after config save: bind")
  })
})
