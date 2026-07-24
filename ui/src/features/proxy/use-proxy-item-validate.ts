import { useCallback } from "react"
import { toast } from "sonner"

import type { ConfigSaveErrorState } from "@/features/config/config-save-error"
import { useConfigQuery } from "@/features/config/config-hooks"
import { useConfigValidate } from "@/features/config/use-config-validate"
import type { JsonValue, SingBoxConfig } from "@/lib/api/types"

type JsonObject = Record<string, JsonValue>

function asObjects(value: JsonValue | undefined): JsonObject[] {
  return Array.isArray(value)
    ? value.filter((item): item is JsonObject => Boolean(item && typeof item === "object" && !Array.isArray(item)))
    : []
}

function defaultReportError(error: unknown): ConfigSaveErrorState {
  const message = error instanceof Error ? error.message : String(error)
  toast.error(message)
  return { message }
}

interface UseProxyItemValidateOptions {
  kind: "inbounds" | "outbounds"
  index: number
  object: JsonObject | null
  reportError?: (error: unknown) => ConfigSaveErrorState
  clearSaveError?: () => void
  onReportedError?: (error: ConfigSaveErrorState) => void
}

/** Dry-run full config with one inbound/outbound draft applied at index. */
export function useProxyItemValidate({
  kind,
  index,
  object,
  reportError,
  clearSaveError,
  onReportedError,
}: UseProxyItemValidateOptions) {
  const configQuery = useConfigQuery()
  const buildConfig = useCallback((): SingBoxConfig | null => {
    if (!object || !configQuery.data) return null
    const next = asObjects(configQuery.data[kind])
    if (index < 0 || index >= next.length) next.push(object)
    else next[index] = object
    return { ...configQuery.data, [kind]: next }
  }, [configQuery.data, index, kind, object])

  const { validating, validate } = useConfigValidate({
    buildConfig,
    reportError: reportError ?? defaultReportError,
    clearSaveError,
    onReportedError,
  })

  return {
    validating,
    validate,
    ready: Boolean(object && configQuery.data),
  }
}
