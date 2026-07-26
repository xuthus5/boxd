import { describe, expect, it } from "vitest"

import { ApiError } from "@/lib/api/client"
import {
  classifyConfigSaveErrorMessage,
  configSaveErrorClipboardText,
  configSaveErrorFromError,
  configSaveErrorFromMessage,
  configSaveErrorFromRollback,
  configSaveErrorHintKey,
  configSectionFromPath,
  configSectionHref,
  configPathEditorHref,
  withConfigPathQuery,
  describeConfigSaveError,
  formatConfigSaveErrorTitle,
  mapConfigApiErrorCode,
} from "@/features/config/config-save-error"

describe("config-save-error", () => {
  it("parses pathful validation messages", () => {
    const state = configSaveErrorFromMessage("inbounds[0].listen_port: invalid")
    expect(state.path).toBe("inbounds[0].listen_port")
    expect(state.message).toContain("inbounds[0].listen_port")
    expect(state.code).toBe("config_invalid")
    expect(state.section).toBe("inbounds")
    expect(configSectionFromPath(state.path)).toBe("inbounds")
  })

  it("handles plain errors, API codes, and rollbacks", () => {
    expect(configSaveErrorFromError(new Error("outbounds[1].server: required")).path).toBe("outbounds[1].server")
    expect(configSaveErrorFromError("plain").message).toContain("plain")
    expect(configSaveErrorFromError(new ApiError("restart failed after config save", 500, "config_restart_failed")).code)
      .toBe("restart_failed")
    expect(configSaveErrorFromRollback({ error: null }, "配置已回滚")).toMatchObject({
      message: "配置已回滚",
      rolledBack: true,
      code: "restart_failed",
    })
    expect(configSaveErrorFromRollback({ error: { code: "x", message: "route.final: missing" } }, "rolled").path)
      .toBe("route.final")
    expect(describeConfigSaveError(null)).toBeNull()
    expect(describeConfigSaveError({ message: "x" })?.summary).toBe("x")
    expect(configSectionFromPath(undefined)).toBeUndefined()
  })

  it("maps codes, hints, clipboard, and section links", () => {
    expect(configSaveErrorHintKey()).toBe("config.errorHintUnknown")
    expect(mapConfigApiErrorCode("config_invalid_runtime")).toBe("config_invalid")
    expect(mapConfigApiErrorCode("config_invalid")).toBe("config_invalid")
    expect(mapConfigApiErrorCode("config_restart_failed")).toBe("restart_failed")
    expect(mapConfigApiErrorCode("restart_failed")).toBe("restart_failed")
    expect(mapConfigApiErrorCode("config_rollback_failed")).toBe("restart_failed")
    expect(mapConfigApiErrorCode("config_missing")).toBe("config_missing")
    expect(mapConfigApiErrorCode("permission")).toBe("permission")
    expect(mapConfigApiErrorCode("forbidden")).toBe("permission")
    expect(mapConfigApiErrorCode("unavailable")).toBe("network")
    expect(mapConfigApiErrorCode("bad_gateway")).toBe("network")
    expect(mapConfigApiErrorCode("request_failed")).toBe("network")
    expect(mapConfigApiErrorCode("  ")).toBeUndefined()
    expect(mapConfigApiErrorCode("")).toBeUndefined()
    expect(mapConfigApiErrorCode("other")).toBeUndefined()
    expect(classifyConfigSaveErrorMessage("Failed to fetch")).toBe("network")
    expect(classifyConfigSaveErrorMessage("no such file")).toBe("config_missing")
    expect(classifyConfigSaveErrorMessage("permission denied")).toBe("permission")
    expect(classifyConfigSaveErrorMessage("restart failed")).toBe("restart_failed")
    expect(classifyConfigSaveErrorMessage("start failed")).toBe("start_failed")
    expect(classifyConfigSaveErrorMessage("invalid config")).toBe("config_invalid")
    expect(classifyConfigSaveErrorMessage("mystery")).toBe("unknown")
    expect(classifyConfigSaveErrorMessage(undefined)).toBe("unknown")
    expect(configSaveErrorHintKey("config_invalid")).toBe("config.errorHintInvalid")
    expect(configSaveErrorHintKey("unknown")).toBe("config.errorHintUnknown")
    expect(configSectionHref("dns")).toBe("/policy/dns")
    expect(configSectionHref("outbounds")).toBe("/proxy/outbounds")
    expect(configSectionHref("ntp")).toBe("/advanced/ntp")
    expect(configSectionHref("inbounds")).toBe("/proxy/inbounds")
    expect(configSectionHref("route")).toBe("/policy/route")
    expect(configSectionHref("experimental")).toBe("/advanced/experimental")
    expect(configSectionHref("endpoints")).toBe("/advanced/endpoints")
    expect(configSectionHref("certificate")).toBe("/advanced/certificate")
    expect(configSectionHref("services")).toBe("/advanced/services")
    expect(configSectionHref("mystery")).toBe("/advanced/raw")
    expect(configPathEditorHref("inbounds[0].listen_port")).toBe(
      "/advanced/raw?path=inbounds%5B0%5D.listen_port",
    )
    expect(withConfigPathQuery("/advanced/raw", "route.final")).toBe("/advanced/raw?path=route.final")
    expect(withConfigPathQuery("/advanced/raw?x=1", "a")).toBe("/advanced/raw?x=1&path=a")
    expect(withConfigPathQuery("/advanced/raw", "  ")).toBe("/advanced/raw")
    expect(configPathEditorHref()).toBe("/advanced/raw")
    expect(configSaveErrorClipboardText({
      message: "inbounds[0].listen_port: invalid",
      path: "inbounds[0].listen_port",
      code: "config_invalid",
      section: "inbounds",
    })).toBe([
      "section: inbounds",
      "path: inbounds[0].listen_port",
      "code: config_invalid",
      "error: inbounds[0].listen_port: invalid",
    ].join("\n"))
    expect(configSaveErrorClipboardText(null)).toBe("")
    expect(configSaveErrorClipboardText({ message: "network failure" })).toContain("code: network")
    expect(configSaveErrorClipboardText({ message: "rolled", code: "restart_failed", rolledBack: true })).toContain("rolled_back: true")
    expect(formatConfigSaveErrorTitle((key, values) => {
      if (key === "config.saveFailedWithCode") return `failed:${values?.code}`
      return key
    }, { message: "x", code: "network" })).toBe("failed:network")
    expect(formatConfigSaveErrorTitle((key) => key, {
      message: "x",
      path: "route.final",
    })).toBe("config.errorPathTitle")
    expect(formatConfigSaveErrorTitle((key) => key, {
      message: "x",
      rolledBack: true,
    })).toBe("config.rolledBackTitle")
    expect(formatConfigSaveErrorTitle((key) => key, {
      message: "x",
      code: "unknown",
    })).toBe("config.saveFailedTitle")
  })

  it("handles empty and non-API error inputs", () => {
    expect(configSaveErrorFromError(new Error("  "), "fallback").message).toBe("fallback")
    expect(configSaveErrorFromError(null, "fallback").message).toBe("fallback")
    expect(configSaveErrorFromError(new ApiError("bad", 400, "config_invalid_runtime")).code).toBe("config_invalid")
    expect(configSaveErrorFromMessage("", { code: "permission", rolledBack: true })).toMatchObject({
      message: "invalid sing-box config",
      code: "permission",
      rolledBack: true,
    })
    expect(configSectionFromPath("  ")).toBeUndefined()
    expect(configSectionFromPath("bad path")).toBe("bad")
    expect(describeConfigSaveError({ message: "" })).toBeNull()
  })
})
