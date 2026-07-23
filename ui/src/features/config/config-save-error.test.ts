import { describe, expect, it } from "vitest"

import {
  configSaveErrorFromError,
  configSaveErrorFromMessage,
  configSaveErrorFromRollback,
  configSectionFromPath,
  describeConfigSaveError,
} from "@/features/config/config-save-error"

describe("config-save-error", () => {
  it("parses pathful validation messages", () => {
    const state = configSaveErrorFromMessage("inbounds[0].listen_port: invalid")
    expect(state.path).toBe("inbounds[0].listen_port")
    expect(state.message).toContain("inbounds[0].listen_port")
    expect(configSectionFromPath(state.path)).toBe("inbounds")
  })

  it("handles plain errors and rollbacks", () => {
    expect(configSaveErrorFromError(new Error("outbounds[1].server: required")).path).toBe("outbounds[1].server")
    expect(configSaveErrorFromError("plain").message).toContain("plain")
    expect(configSaveErrorFromRollback({ error: null }, "配置已回滚").message).toBe("配置已回滚")
    expect(configSaveErrorFromRollback({ error: { code: "x", message: "route.final: missing" } }, "rolled").path)
      .toBe("route.final")
    expect(describeConfigSaveError(null)).toBeNull()
    expect(describeConfigSaveError({ message: "x" })?.summary).toBe("x")
    expect(configSectionFromPath(undefined)).toBeUndefined()
  })
})
