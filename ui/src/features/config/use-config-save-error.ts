import { useCallback, useState } from "react"
import { toast } from "sonner"

import {
  configSaveErrorFromError,
  configSaveErrorFromRollback,
  type ConfigSaveErrorState,
} from "@/features/config/config-save-error"
import type { APIEnvelope } from "@/lib/api/types"

function toastMessage(state: ConfigSaveErrorState) {
  if (state.code && state.code !== "unknown") {
    return `${state.message} (${state.code})`
  }
  return state.message
}

export function useConfigSaveError() {
  const [saveError, setSaveError] = useState<ConfigSaveErrorState | null>(null)
  const clearSaveError = useCallback(() => setSaveError(null), [])

  const reportError = useCallback((error: unknown) => {
    const next = configSaveErrorFromError(error)
    setSaveError(next)
    toast.error(toastMessage(next))
    return next
  }, [])

  const reportRollback = useCallback((response: Pick<APIEnvelope<unknown>, "error">, fallback: string) => {
    const next = configSaveErrorFromRollback(response, fallback)
    setSaveError(next)
    toast.error(toastMessage(next))
    return next
  }, [])

  return { saveError, setSaveError, clearSaveError, reportError, reportRollback }
}
