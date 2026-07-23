import { useTranslation } from "react-i18next"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import type { ConfigSaveErrorState } from "@/features/config/config-save-error"

interface ConfigSaveErrorAlertProps {
  error: ConfigSaveErrorState | null
  onDismiss?: () => void
  onJumpToPath?: (path: string) => void
  className?: string
}

export function ConfigSaveErrorAlert({
  error,
  onDismiss,
  onJumpToPath,
  className,
}: ConfigSaveErrorAlertProps) {
  const { t } = useTranslation()
  if (!error) return null
  return (
    <Alert variant="destructive" className={className} data-testid="config-save-error">
      <AlertTitle>
        {error.path ? t("config.errorPathTitle") : t("config.saveFailedTitle")}
      </AlertTitle>
      <AlertDescription className="flex flex-col gap-2">
        <span>
          {error.path
            ? t("config.errorPathDescription", { path: error.path })
            : error.message}
        </span>
        {error.path && error.message !== error.path ? (
          <span className="text-sm opacity-90">{error.message}</span>
        ) : null}
        <div className="flex flex-wrap gap-2">
          {error.path && onJumpToPath ? (
            <Button type="button" size="xs" variant="outline" onClick={() => onJumpToPath(error.path!)}>
              {t("config.jumpToPath")}
            </Button>
          ) : null}
          {onDismiss ? (
            <Button type="button" size="xs" variant="ghost" onClick={onDismiss}>
              {t("common.dismiss")}
            </Button>
          ) : null}
        </div>
      </AlertDescription>
    </Alert>
  )
}
