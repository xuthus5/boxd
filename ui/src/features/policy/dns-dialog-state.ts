import { transformDNSField } from "@/features/policy/dns-form-model"
import { usePolicyDialogState } from "@/features/policy/policy-dialog-state"
import type { JsonObject } from "@/features/policy/policy-form-model"

export function optionsWithCurrent(values: readonly string[], current: string): string[] {
  return current && !values.includes(current) ? [current, ...values] : [...values]
}

export function useDNSDialogState(item: JsonObject) {
  return usePolicyDialogState(item, transformDNSField)
}
