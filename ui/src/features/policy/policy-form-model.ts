import type { JsonValue } from "@/lib/api/types"

export type JsonObject = Record<string, JsonValue>
export type PolicySection = "route" | "dns"
export type PolicyFieldKind =
  | "text" | "textarea" | "number" | "boolean" | "list" | "number-list"
  | "select" | "json-object" | "json-array" | "ref" | "ref-multi" | "network-multi" | "network-interface"

export type PolicyFieldRef = "inbound" | "outbound" | "dns-server" | "rule-set" | "network-interface"

export interface PolicyFieldWhen {
  path: string
  is?: JsonValue | readonly JsonValue[]
  falsy?: boolean
}

export interface PolicyFieldSpec {
  path: string
  label: string
  kind?: PolicyFieldKind
  options?: readonly string[]
  required?: boolean
  section?: string
  when?: PolicyFieldWhen | readonly PolicyFieldWhen[]
  ref?: PolicyFieldRef
}

export interface PolicyFormContext {
  inboundTags?: string[]
  outboundTags?: string[]
  dnsServerTags?: string[]
  ruleSetTags?: string[]
  currentTag?: string
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

function isFalsy(value: JsonValue | undefined) {
  return value === undefined || value === null || value === false || value === "" || (Array.isArray(value) && value.length === 0)
}

function matchesWhen(object: JsonObject, rule: PolicyFieldWhen) {
  const value = getPolicyPath(object, rule.path)
  if (rule.falsy) return isFalsy(value)
  if (rule.is !== undefined) {
    const allowed = Array.isArray(rule.is) ? rule.is : [rule.is]
    return allowed.some((item) => Object.is(item, value))
  }
  return !isFalsy(value)
}

export function isPolicyFieldVisible(object: JsonObject, field: PolicyFieldSpec) {
  if (!field.when) return true
  const rules = Array.isArray(field.when) ? field.when : [field.when]
  return rules.every((rule) => matchesWhen(object, rule))
}

export function visiblePolicyFields(fields: readonly PolicyFieldSpec[], object: JsonObject) {
  return fields.filter((field) => isPolicyFieldVisible(object, field))
}

export function groupPolicyFieldsBySection(fields: readonly PolicyFieldSpec[]) {
  const groups: { section?: string; fields: PolicyFieldSpec[] }[] = []
  for (const field of fields) {
    const last = groups[groups.length - 1]
    if (last && last.section === field.section) last.fields.push(field)
    else groups.push({ section: field.section, fields: [field] })
  }
  return groups
}

export function pruneInvisiblePolicyFields(object: JsonObject, fields: readonly PolicyFieldSpec[]) {
  let next = object
  let changed = true
  while (changed) {
    changed = false
    for (const field of fields) {
      if (isPolicyFieldVisible(next, field)) continue
      if (getPolicyPath(next, field.path) === undefined) continue
      next = setPolicyPath(next, field.path, undefined)
      changed = true
    }
  }
  return next
}

export function policyConfigTags(items: unknown, currentTag?: string) {
  if (!Array.isArray(items)) return [] as string[]
  const tags = items.flatMap((item) => {
    if (!item || typeof item !== "object" || Array.isArray(item)) return []
    const tag = (item as JsonObject).tag
    return typeof tag === "string" && tag && tag !== currentTag ? [tag] : []
  })
  return [...new Set(tags)]
}

export function policyDNSServerTags(dns: unknown) {
  if (!dns || typeof dns !== "object" || Array.isArray(dns)) return [] as string[]
  return policyConfigTags((dns as JsonObject).servers)
}

export function policyRuleSetTags(route: unknown) {
  if (!route || typeof route !== "object" || Array.isArray(route)) return [] as string[]
  return policyConfigTags((route as JsonObject).rule_set)
}
