import { RotateCcwIcon } from "lucide-react"
import { useTranslation } from "react-i18next"

import { ConfirmAction } from "@/components/confirm-action"
import { Button } from "@/components/ui/button"
import { shortConfigHash } from "@/features/dashboard/config-apply-source"
import type { ConfigApplyEvent } from "@/lib/api/types"

interface ConfigRestoreActionProps {
  event: ConfigApplyEvent
  restoring: boolean
  onRestore: (event: ConfigApplyEvent) => Promise<void>
}

export function ConfigRestoreAction({ event, restoring, onRestore }: ConfigRestoreActionProps) {
  const { t } = useTranslation()
  return (
    <div className="mt-1.5 flex justify-end">
      <ConfirmAction
        trigger={(
          <Button
            type="button"
            size="sm"
            variant="outline"
            className="h-7"
            disabled={restoring}
            aria-busy={restoring}
          >
            <RotateCcwIcon data-icon="inline-start" />
            {restoring ? t("configRestore.restoringConfig") : t("configRestore.restoreConfig")}
          </Button>
        )}
        title={t("configRestore.restoreConfigTitle")}
        description={t("configRestore.restoreConfigDescription", { hash: shortConfigHash(event.hash) })}
        confirmLabel={t("configRestore.confirmRestoreConfig")}
        confirmVariant="outline"
        onConfirm={() => onRestore(event)}
      />
    </div>
  )
}
