import { isJsonObject, type JsonObject, type PolicyFieldSpec } from "@/features/policy/policy-form-model"
import type { JsonValue, SingBoxConfig } from "@/lib/api/types"

export const logLevels = ["trace", "debug", "info", "warn", "error", "fatal", "panic"] as const
export type LogLevel = (typeof logLevels)[number]

export const logFields = [
  { path: "disabled", label: "disabled", kind: "boolean", section: "basic" },
  { path: "level", label: "level", kind: "select", options: logLevels, section: "basic" },
  { path: "output", label: "output", section: "output" },
  { path: "timestamp", label: "timestamp", kind: "boolean", section: "output" },
] as const satisfies readonly PolicyFieldSpec[]

function isOptionalBoolean(value: JsonValue | undefined): boolean {
  return value === undefined || typeof value === "boolean"
}

function isOptionalString(value: JsonValue | undefined): boolean {
  return value === undefined || typeof value === "string"
}

function isSupportedLevel(value: JsonValue | undefined): boolean {
  if (value === undefined) return true
  if (typeof value !== "string") return false
  const level = value.trim()
  return level === "" || logLevels.includes(level as LogLevel)
}

export function isLogStructureValid(value: JsonValue | null | undefined): value is JsonObject {
  const object = value ?? undefined
  if (!isJsonObject(object)) return false
  return isOptionalBoolean(object.disabled)
    && isSupportedLevel(object.level)
    && isOptionalString(object.output)
    && isOptionalBoolean(object.timestamp)
}

export function normalizeLogObject(value: JsonValue | undefined): JsonObject {
  return isJsonObject(value) ? { ...value } : {}
}

function trimKnownString(object: JsonObject, path: "level" | "output") {
  const value = object[path]
  if (typeof value !== "string" || !value.trim()) {
    delete object[path]
    return
  }
  object[path] = value.trim()
}

export function prepareLogObject(object: JsonObject): JsonObject {
  const prepared = normalizeLogObject(object)
  if (prepared.disabled !== true) delete prepared.disabled
  if (prepared.timestamp !== true) delete prepared.timestamp
  trimKnownString(prepared, "level")
  trimKnownString(prepared, "output")
  return prepared
}

export function applyLogConfig(config: SingBoxConfig, object: JsonObject): SingBoxConfig {
  const next = { ...config }
  const log = prepareLogObject(object)
  if (Object.keys(log).length === 0) delete next.log
  else next.log = log
  return next
}
