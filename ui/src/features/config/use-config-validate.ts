import { useQueryClient } from "@tanstack/react-query"
import { useCallback, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import type { ConfigSaveErrorState } from "@/features/config/config-save-error"
import { api } from "@/lib/api/endpoints"
import type { SingBoxConfig } from "@/lib/api/types"

/** Known dry-run entry labels mirrored by the apply timeline. */
export type ConfigValidateSource =
  | "validate"
  | "validate_raw"
  | "validate_endpoints"
  | "validate_certificate"
  | "validate_ntp"
  | "validate_experimental"
  | "validate_inbounds"
  | "validate_outbounds"
  | "validate_route"
  | "validate_dns"

interface UseConfigValidateOptions {
  /** Build the full config document to dry-run. Return null to skip. */
  buildConfig: () => SingBoxConfig | null
  reportError: (error: unknown) => ConfigSaveErrorState
  clearSaveError?: () => void
  /** Optional side effect after densified report (e.g. path reveal). */
  onReportedError?: (error: ConfigSaveErrorState) => void
  /** Editor entry label for the apply timeline. */
  source?: ConfigValidateSource
}

/** Dry-run sing-box config validation without writing or restarting. */
export function useConfigValidate({
  buildConfig,
  reportError,
  clearSaveError,
  onReportedError,
  source,
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
      await api.config.validate(config, source ? { source } : undefined)
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
  }, [buildConfig, clearSaveError, onReportedError, queryClient, reportError, source, t])

  return { validating, validate }
}
