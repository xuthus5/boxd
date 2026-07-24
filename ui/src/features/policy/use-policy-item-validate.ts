import { useCallback } from "react"

import type { ConfigSaveErrorState } from "@/features/config/config-save-error"
import { reportConfigValidateError } from "@/features/config/report-config-validate-error"
import { useConfigQuery } from "@/features/config/config-hooks"
import { useConfigValidate } from "@/features/config/use-config-validate"
import { isJsonObject, type JsonObject } from "@/features/policy/policy-form-model"
import {
  parsePolicyItemPath,
  type PolicyItemKind,
  type PolicySectionPath,
} from "@/features/policy/policy-path"
import type { JsonValue, SingBoxConfig } from "@/lib/api/types"

function asObjects(value: JsonValue | undefined): JsonObject[] {
  return Array.isArray(value)
    ? value.filter((item): item is JsonObject => Boolean(item && typeof item === "object" && !Array.isArray(item)))
    : []
}

/** Relative path for the open policy item dialog, if the error path targets it. */
export function policyItemErrorRelativePath(
  path: string | undefined,
  section: PolicySectionPath,
  kind: PolicyItemKind,
  index: number,
): string | undefined {
  if (!path) return undefined
  const target = parsePolicyItemPath(path, section)
  if (!target || target.kind !== kind) return undefined
  if (index >= 0 && target.index !== index) return undefined
  return target.relativePath || undefined
}

interface UsePolicyItemValidateOptions {
  section: PolicySectionPath
  kind: PolicyItemKind
  index: number
  object: JsonObject | null
  reportError?: (error: unknown) => ConfigSaveErrorState
  clearSaveError?: () => void
  onReportedError?: (error: ConfigSaveErrorState) => void
}

/** Dry-run full config with one route/dns list item draft applied. */
export function usePolicyItemValidate({
  section,
  kind,
  index,
  object,
  reportError,
  clearSaveError,
  onReportedError,
}: UsePolicyItemValidateOptions) {
  const configQuery = useConfigQuery()
  const buildConfig = useCallback((): SingBoxConfig | null => {
    if (!object || !configQuery.data) return null
    const sectionValue = configQuery.data[section]
    const sectionObject = isJsonObject(sectionValue) ? { ...sectionValue } : {}
    const next = asObjects(sectionObject[kind])
    if (index < 0 || index >= next.length) next.push(object)
    else next[index] = object
    return { ...configQuery.data, [section]: { ...sectionObject, [kind]: next } }
  }, [configQuery.data, index, kind, object, section])

  const { validating, validate } = useConfigValidate({
    buildConfig,
    reportError: reportError ?? reportConfigValidateError,
    clearSaveError,
    onReportedError,
    source: section === "dns" ? "validate_dns" : "validate_route",
  })

  return {
    validating,
    validate,
    ready: Boolean(object && configQuery.data),
  }
}
