import type { JsonValue } from "@/lib/api/types"

export type JsonObject = Record<string, JsonValue>
export type PolicySection = "route" | "dns"
export type PolicyFieldKind = "text" | "textarea" | "number" | "boolean" | "list" | "number-list" | "select" | "json-object" | "json-array"

export interface PolicyFieldSpec {
  path: string
  label: string
  kind?: PolicyFieldKind
  options?: readonly string[]
  required?: boolean
}

export type PolicyFieldTransform = (object: JsonObject, field: PolicyFieldSpec, raw: string) => JsonObject | null | undefined

export function isJsonObject(value: JsonValue | undefined): value is JsonObject {
  return value !== null && typeof value === "object" && !Array.isArray(value)
}

export function isNonEmptyJsonObjectArray(value: JsonValue | undefined): value is JsonObject[] {
  return Array.isArray(value) && value.length > 0 && value.every(isJsonObject)
}

const sectionArrayPaths: Record<PolicySection, readonly string[]> = {
  route: ["rules", "rule_set"],
  dns: ["servers", "rules"],
}

export function isPolicySectionStructureValid(section: PolicySection, object: JsonObject): boolean {
  return sectionArrayPaths[section].every((path) => {
    const value = object[path]
    return value === undefined || Array.isArray(value) && value.every(isJsonObject)
  })
}

export function getPolicyPath(object: JsonObject, path: string): JsonValue | undefined {
  return path.split(".").reduce<JsonValue | undefined>((value, key) => isJsonObject(value) ? value[key] : undefined, object)
}

export function setPolicyPath(object: JsonObject, path: string, value: JsonValue | undefined): JsonObject {
  const keys = path.split(".")
  const update = (source: JsonObject, index: number): JsonObject => {
    const next = { ...source }
    const key = keys[index]
    if (index === keys.length - 1) {
      if (value === undefined) delete next[key]
      else next[key] = value
      return next
    }

    const child = isJsonObject(next[key]) ? next[key] : {}
    const updated = update(child, index + 1)
    if (Object.keys(updated).length > 0) next[key] = updated
    else delete next[key]
    return next
  }

  return update(object, 0)
}

export function moveItem<T>(items: readonly T[], index: number, direction: -1 | 1): T[] {
  const target = index + direction
  if (index < 0 || index >= items.length || target < 0 || target >= items.length) return [...items]
  const next = [...items]
  ;[next[index], next[target]] = [next[target], next[index]]
  return next
}

export function cloneJsonObject(object: JsonObject): JsonObject {
  return structuredClone(object)
}
