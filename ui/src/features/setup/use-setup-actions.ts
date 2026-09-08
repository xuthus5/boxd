import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { useConfigSaveError } from "@/features/config/use-config-save-error"
import { setupAPI, type SetupInput, type SetupPreview, type SetupStatus } from "@/lib/api/setup"

type SaveErrors = ReturnType<typeof useConfigSaveError>

function useSetupApply(options: {
  errors: SaveErrors
  resetPreview: () => void
  onClose: () => void
  willRestart: boolean
}) {
  const client = useQueryClient()
  const { t } = useTranslation()
  return useMutation({
    mutationFn: setupAPI.apply,
    onSuccess: (response) => {
      options.resetPreview()
      if (response.status === "rolled_back") {
        options.errors.reportRollback(response, t("setup.rolledBack"))
        return
      }
      toast.success(t(options.willRestart ? "setup.appliedRunning" : "setup.appliedStopped"))
      options.onClose()
    },
    onError: (error) => { options.resetPreview(); options.errors.reportError(error) },
    onSettled: () => Promise.all([
      client.invalidateQueries({ queryKey: ["config"] }),
      client.invalidateQueries({ queryKey: ["service"] }),
      client.invalidateQueries({ queryKey: ["nodes"] }),
    ]),
  })
}

export function useSetupActions(status: SetupStatus, onClose: () => void) {
  const [input, setInput] = useState<SetupInput>(() => ({
    modules: status.modules.map((item) => item.id),
    inbound_mode: status.listeners.length ? "preserve" : "proxy",
    ipv6: "auto",
  }))
  const [preview, setPreview] = useState<SetupPreview | null>(null)
  const errors = useConfigSaveError()
  const previewMutation = useMutation({ mutationFn: setupAPI.preview, onSuccess: setPreview, onError: errors.reportError })
  const applyMutation = useSetupApply({ errors, resetPreview: () => setPreview(null), onClose, willRestart: preview?.will_restart === true })
  const change = (next: SetupInput) => { setInput(next); setPreview(null); errors.clearSaveError() }
  const makePreview = () => { setPreview(null); errors.clearSaveError(); previewMutation.mutate(input) }
  const apply = () => { if (preview) { errors.clearSaveError(); applyMutation.mutate({ ...input, source_hash: preview.source_hash }) } }
  return { input, preview, errors, change, makePreview, apply,
    previewing: previewMutation.isPending, applying: applyMutation.isPending,
    busy: previewMutation.isPending || applyMutation.isPending,
    canPreview: input.modules.length > 0 && (!status.config_error || input.reset_invalid === true),
  }
}
