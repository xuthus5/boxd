import type { JsonObject } from "@/features/policy/policy-form-model"

export function isRuleInverted(item: JsonObject) {
  return item.invert === true
}

export function toggleRuleInvert(item: JsonObject): JsonObject {
  if (item.invert === true) {
    const next = { ...item }
    delete next.invert
    return next
  }
  return { ...item, invert: true }
}
