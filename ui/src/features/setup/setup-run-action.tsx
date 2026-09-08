import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { ConfigSaveErrorAlert } from "@/features/config/config-save-error-alert"
import { useConfigSaveError } from "@/features/config/use-config-save-error"
import { api } from "@/lib/api/endpoints"

export function SetupRunAction({ running, ready }: { running: boolean; ready: boolean }) {
  const { t } = useTranslation()
  const client = useQueryClient()
  const errors = useConfigSaveError()
  const action = useMutation({
    mutationFn: async () => {
      errors.clearSaveError()
      const config = await api.config.get()
      await api.config.validate(config, { source: "setup" })
      if (!running) await api.service.start()
    },
    onSuccess: () => { toast.success(t(running ? "setup.validated" : "setup.started")) },
    onError: errors.reportError,
    onSettled: () => Promise.all([
      client.invalidateQueries({ queryKey: ["service"] }),
      client.invalidateQueries({ queryKey: ["config"] }),
    ]),
  })
  return <div className="flex flex-col gap-2">
    <Button variant="outline" size="sm" disabled={action.isPending || !ready} onClick={() => action.mutate()}>
      {t(action.isPending ? "setup.validating" : running ? "setup.validate" : "setup.validateStart")}
    </Button>
    <ConfigSaveErrorAlert error={errors.saveError} onDismiss={errors.clearSaveError} />
  </div>
}
