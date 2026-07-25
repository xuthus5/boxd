import { useQueryClient } from "@tanstack/react-query"
import { useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import {
  configSaveErrorFromError,
  configSaveErrorFromRollback,
} from "@/features/config/config-save-error"
import { reportConfigSaveErrorToast } from "@/features/config/config-save-error-actions"
import { api } from "@/lib/api/endpoints"
import type { ConfigApplyEvent } from "@/lib/api/types"

export function useConfigRestore() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const inFlight = useRef(false)
  const [restoringID, setRestoringID] = useState<string | null>(null)
  const restore = async (event: ConfigApplyEvent) => {
    if (!event.id || inFlight.current) return
    inFlight.current = true
    setRestoringID(event.id)
    try {
      const response = await api.config.restoreApplyHistory(event.id)
      if (response.data.already_current) {
        toast.success(t("configRestore.alreadyCurrent"))
        return
      }
      await Promise.allSettled([
        queryClient.invalidateQueries({ queryKey: ["config"] }),
        queryClient.invalidateQueries({ queryKey: ["service"] }),
      ])
      if (response.status === "rolled_back") {
        reportConfigSaveErrorToast(configSaveErrorFromRollback(response, t("configRestore.restoreFailed")))
        return
      }
      toast.success(t("configRestore.restoreSuccess"))
    } catch (error) {
      reportConfigSaveErrorToast(configSaveErrorFromError(error, t("configRestore.restoreFailed")))
    } finally {
      inFlight.current = false
      setRestoringID(null)
    }
  }
  return { restore, restoringID }
}
