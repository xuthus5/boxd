import { CopyIcon } from "lucide-react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  classifyPageLoadError,
  formatPageLoadErrorTitle,
  pageLoadErrorClipboardText,
  pageLoadErrorHintKey,
  pageLoadErrorMessage,
} from "@/features/common/page-load-error"
import { copyText } from "@/lib/clipboard"

interface PageLoadErrorAlertProps {
  error: unknown
  /** Clipboard/debug scope, e.g. dashboard / nodes. */
  scope?: string
  /** Optional request path for diagnostics. */
  path?: string
  /** i18n title key; defaults to common.loadFailed. */
  titleKey?: string
  className?: string
  onRetry?: () => void
}

export function PageLoadErrorAlert({
  error,
  scope,
  path,
  titleKey = "common.loadFailed",
  className,
  onRetry,
}: PageLoadErrorAlertProps) {
  const { t } = useTranslation()
  const code = classifyPageLoadError(error)
  const message = pageLoadErrorMessage(error, t(titleKey))
  const payload = pageLoadErrorClipboardText(error, { scope, path })
  const codeLabel = code !== "unknown" ? code : undefined

  return (
    <Alert
      variant="destructive"
      className={className}
      data-testid="page-load-error"
      data-error-code={code}
    >
      <div className="flex items-start justify-between gap-2">
        <AlertTitle>{formatPageLoadErrorTitle(t, code, titleKey)}</AlertTitle>
        <Button
          type="button"
          variant="ghost"
          size="xs"
          className="h-6 shrink-0 px-1.5 text-destructive"
          aria-label={t("common.copyLoadError")}
          onClick={() => {
            if (!payload) return
            void copyText(payload).then(
              () => toast.success(t("common.loadErrorCopied")),
              () => toast.error(t("common.loadErrorCopyFailed")),
            )
          }}
        >
          <CopyIcon className="size-3.5" aria-hidden="true" />
        </Button>
      </div>
      <AlertDescription className="mt-1 flex flex-col gap-2">
        {codeLabel ? (
          <Badge variant="outline" className="w-fit font-mono text-[10px]">
            {codeLabel}
          </Badge>
        ) : null}
        <span className="break-words">{message}</span>
        <span className="text-[11px] text-muted-foreground">{t(pageLoadErrorHintKey(code))}</span>
        {onRetry ? (
          <div className="flex flex-wrap gap-2">
            <Button type="button" size="xs" variant="outline" onClick={onRetry}>
              {t("common.retry")}
            </Button>
          </div>
        ) : null}
      </AlertDescription>
    </Alert>
  )
}
