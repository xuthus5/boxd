import { CopyIcon } from "lucide-react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  classifyPageLoadError,
  pageLoadErrorClipboardText,
  pageLoadErrorHintKey,
  pageLoadErrorMessage,
} from "@/features/common/page-load-error"
import { copyText } from "@/features/proxy/copy-tag-button"

interface CardQueryErrorProps {
  error: unknown
  scope: string
  path?: string
  /** Fallback when error has no message. */
  fallback?: string
  className?: string
  onRetry?: () => void
}

/** Compact densified diagnostics for card-level query failures. */
export function CardQueryError({
  error,
  scope,
  path,
  fallback,
  className,
  onRetry,
}: CardQueryErrorProps) {
  const { t } = useTranslation()
  const code = classifyPageLoadError(error)
  const message = pageLoadErrorMessage(error, fallback ?? t("common.loadFailed"))
  const payload = pageLoadErrorClipboardText(error, { scope, path })
  const codeLabel = code !== "unknown" ? code : undefined

  return (
    <div
      className={className}
      data-testid="card-query-error"
      data-error-code={code}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0 flex flex-col gap-1">
          {codeLabel ? (
            <Badge variant="outline" className="w-fit font-mono text-[10px]">
              {codeLabel}
            </Badge>
          ) : null}
          <p className="break-words text-sm text-destructive" title={message}>
            {message}
          </p>
          <p className="text-[11px] text-muted-foreground">
            {t(pageLoadErrorHintKey(code))}
          </p>
        </div>
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
      {onRetry ? (
        <div className="mt-2">
          <Button type="button" size="xs" variant="outline" onClick={onRetry}>
            {t("common.retry")}
          </Button>
        </div>
      ) : null}
    </div>
  )
}
