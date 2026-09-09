import type { JsonObject } from "@/features/policy/policy-form-model"

export function hasRuleMatchConditions(rule: JsonObject): boolean {
  return Object.entries(rule).some(([key, value]) => {
    if (key === "type" || key === "invert") return false
    if (Array.isArray(value)) return value.length > 0
    if (value && typeof value === "object") return Object.keys(value).length > 0
    if (typeof value === "number") return Number.isFinite(value)
    return Boolean(value)
  })
}
