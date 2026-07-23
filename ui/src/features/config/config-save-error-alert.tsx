import { CopyIcon } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
import {
  configSaveErrorClipboardText,
  configSaveErrorHintKey,
  configSectionHref,
  formatConfigSaveErrorTitle,
  type ConfigSaveErrorState,
} from "@/features/config/config-save-error"
import { copyText } from "@/features/proxy/copy-tag-button"
import { cn } from "@/lib/utils"

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
  const code = error.code && error.code !== "unknown" ? error.code : undefined
  const sectionHref = configSectionHref(error.section)
  const payload = configSaveErrorClipboardText(error)

  return (
    <Alert variant="destructive" className={className} data-testid="config-save-error" data-error-code={error.code || "unknown"}>
      <div className="flex items-start justify-between gap-2">
        <AlertTitle>{formatConfigSaveErrorTitle(t, error)}</AlertTitle>
        <Button
          type="button"
          variant="ghost"
          size="xs"
          className="h-6 shrink-0 px-1.5 text-destructive"
          aria-label={t("config.copySaveError")}
          onClick={() => {
            if (!payload) return
            void copyText(payload).then(
              () => toast.success(t("config.saveErrorCopied")),
              () => toast.error(t("config.saveErrorCopyFailed")),
            )
          }}
        >
          <CopyIcon className="size-3.5" aria-hidden="true" />
        </Button>
      </div>
      <AlertDescription className="mt-1 flex flex-col gap-2">
        {code ? <Badge variant="outline" className="w-fit font-mono text-[10px]">{code}</Badge> : null}
        <span>
          {error.path
            ? t("config.errorPathDescription", { path: error.path })
            : error.message}
        </span>
        {error.path && error.message !== error.path ? (
          <span className="text-sm opacity-90">{error.message}</span>
        ) : null}
        <span className="text-[11px] text-muted-foreground">{t(configSaveErrorHintKey(error.code))}</span>
        <div className="flex flex-wrap gap-2">
          {error.path && onJumpToPath ? (
            <Button type="button" size="xs" variant="outline" onClick={() => onJumpToPath(error.path!)}>
              {t("config.jumpToPath")}
            </Button>
          ) : null}
          {error.section ? (
            <Link
              to={sectionHref}
              className={cn(buttonVariants({ variant: "outline", size: "xs" }), "h-7")}
            >
              {t("config.openSection")}
            </Link>
          ) : null}
          <Link
            to="/"
            className={cn(buttonVariants({ variant: "outline", size: "xs" }), "h-7")}
          >
            {t("config.openApplyTimeline")}
          </Link>
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
