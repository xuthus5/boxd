import { useQueryClient } from "@tanstack/react-query"
import { useCallback, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import type { ConfigSaveErrorState } from "@/features/config/config-save-error"
import { api } from "@/lib/api/endpoints"
import type { SingBoxConfig } from "@/lib/api/types"

interface UseConfigValidateOptions {
  /** Build the full config document to dry-run. Return null to skip. */
  buildConfig: () => SingBoxConfig | null
  reportError: (error: unknown) => ConfigSaveErrorState
  clearSaveError?: () => void
  /** Optional side effect after densified report (e.g. path reveal). */
  onReportedError?: (error: ConfigSaveErrorState) => void
}

/** Dry-run sing-box config validation without writing or restarting. */
export function useConfigValidate({
  buildConfig,
  reportError,
  clearSaveError,
  onReportedError,
}: UseConfigValidateOptions) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [validating, setValidating] = useState(false)

  const validate = useCallback(async () => {
    const config = buildConfig()
    if (!config) return false
    clearSaveError?.()
    setValidating(true)
    try {
      await api.config.validate(config)
      toast.success(t("advanced.validateOK"))
      return true
    } catch (error) {
      const next = reportError(error)
      onReportedError?.(next)
      return false
    } finally {
      setValidating(false)
      void queryClient.invalidateQueries({ queryKey: ["config", "apply-history"] })
    }
  }, [buildConfig, clearSaveError, onReportedError, queryClient, reportError, t])

  return { validating, validate }
}
