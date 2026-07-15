import type { JsonValue } from "@/lib/api/types"

export type JsonObject = Record<string, JsonValue>
export type FieldKind = "text" | "textarea" | "number" | "boolean" | "list" | "number-list" | "select" | "json-object" | "users"
export interface FieldSpec { path: string; label: string; kind?: FieldKind; options?: string[] }
export type FieldTransform = (object: JsonObject, field: FieldSpec, raw: string) => JsonObject | null | undefined

export function getPath(object: JsonObject, path: string): JsonValue | undefined {
  return path.split(".").reduce<JsonValue | undefined>((value, key) => value && typeof value === "object" && !Array.isArray(value) ? value[key] : undefined, object)
}

export function setPath(object: JsonObject, path: string, value: JsonValue | undefined): JsonObject {
  const keys = path.split(".")
  const update = (source: JsonObject, index: number): JsonObject => {
    const next = { ...source }
    const key = keys[index]
    if (index === keys.length - 1) {
      if (value === undefined) delete next[key]
      else next[key] = value
    } else {
      const child = next[key]
      const updated = update(child && typeof child === "object" && !Array.isArray(child) ? child : {}, index + 1)
      if (Object.keys(updated).length) next[key] = updated; else delete next[key]
    }
    return next
  }
  return update(object, 0)
}
