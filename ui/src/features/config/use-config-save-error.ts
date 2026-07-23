import { useCallback, useState } from "react"

import { reportConfigSaveErrorToast } from "@/features/config/config-save-error-actions"
import {
  configSaveErrorFromError,
  configSaveErrorFromRollback,
  type ConfigSaveErrorState,
} from "@/features/config/config-save-error"
import type { APIEnvelope } from "@/lib/api/types"

export function useConfigSaveError() {
  const [saveError, setSaveError] = useState<ConfigSaveErrorState | null>(null)
  const clearSaveError = useCallback(() => setSaveError(null), [])

  const reportError = useCallback((error: unknown) => {
    const next = configSaveErrorFromError(error)
    setSaveError(next)
    reportConfigSaveErrorToast(next)
    return next
  }, [])

  const reportRollback = useCallback((response: Pick<APIEnvelope<unknown>, "error">, fallback: string) => {
    const next = configSaveErrorFromRollback(response, fallback)
    setSaveError(next)
    reportConfigSaveErrorToast(next)
    return next
  }, [])

  return { saveError, setSaveError, clearSaveError, reportError, reportRollback }
}
