import type { JsonValue } from "@/lib/api/types"

export type JsonObject = Record<string, JsonValue>
export type FieldKind =
  | "text" | "textarea" | "number" | "boolean" | "list" | "number-list"
  | "select" | "json-object" | "users" | "listen-address" | "network-interface"
  | "ref" | "network-multi"

export type FieldRef = "inbound" | "dns-server" | "outbound" | "network-interface-multi"

export interface FieldWhen {
  path: string
  is?: JsonValue | readonly JsonValue[]
  falsy?: boolean
}

export interface FieldSpec {
  path: string
  label: string
  kind?: FieldKind
  options?: string[]
  when?: FieldWhen | FieldWhen[]
  section?: string
  ref?: FieldRef
}

export interface FormFieldContext {
  inboundTags?: string[]
  outboundTags?: string[]
  dnsServerTags?: string[]
  currentTag?: string
  inboundType?: string
}

export type FieldTransform = (object: JsonObject, field: FieldSpec, raw: string) => JsonObject | null | undefined

export function getPath(object: JsonObject, path: string): JsonValue | undefined {
  return path.split(".").reduce<JsonValue | undefined>((value, key) => (
    value && typeof value === "object" && !Array.isArray(value) ? value[key] : undefined
  ), object)
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
      const updated = update(child && typeof child === "object" && !Array.isArray(child) ? child as JsonObject : {}, index + 1)
      if (Object.keys(updated).length) next[key] = updated
      else delete next[key]
    }
    return next
  }
  return update(object, 0)
}

function isFalsy(value: JsonValue | undefined) {
  return value === undefined || value === null || value === false || value === "" || (Array.isArray(value) && value.length === 0)
}

function matchesWhen(object: JsonObject, rule: FieldWhen) {
  const value = getPath(object, rule.path)
  if (rule.falsy) return isFalsy(value)
  if (rule.is !== undefined) {
    const allowed = Array.isArray(rule.is) ? rule.is : [rule.is]
    return allowed.some((item) => Object.is(item, value))
  }
  return !isFalsy(value)
}

export function isFieldVisible(object: JsonObject, field: FieldSpec) {
  if (!field.when) return true
  const rules = Array.isArray(field.when) ? field.when : [field.when]
  return rules.every((rule) => matchesWhen(object, rule))
}

export function visibleFields(fields: FieldSpec[], object: JsonObject) {
  return fields.filter((field) => isFieldVisible(object, field))
}

export function groupFieldsBySection(fields: FieldSpec[]) {
  const groups: { section?: string; fields: FieldSpec[] }[] = []
  for (const field of fields) {
    const last = groups[groups.length - 1]
    if (last && last.section === field.section) last.fields.push(field)
    else groups.push({ section: field.section, fields: [field] })
  }
  return groups
}

export function pruneInvisibleFields(object: JsonObject, fields: FieldSpec[]) {
  let next = object
  let changed = true
  while (changed) {
    changed = false
    for (const field of fields) {
      if (isFieldVisible(next, field)) continue
      if (getPath(next, field.path) === undefined) continue
      next = setPath(next, field.path, undefined)
      changed = true
    }
  }
  return next
}

export function configTags(items: unknown, currentTag?: string) {
  if (!Array.isArray(items)) return [] as string[]
  const tags = items.flatMap((item) => {
    if (!item || typeof item !== "object" || Array.isArray(item)) return []
    const tag = (item as JsonObject).tag
    return typeof tag === "string" && tag && tag !== currentTag ? [tag] : []
  })
  return [...new Set(tags)]
}

export function dnsServerTags(dns: unknown) {
  if (!dns || typeof dns !== "object" || Array.isArray(dns)) return [] as string[]
  return configTags((dns as JsonObject).servers)
}
